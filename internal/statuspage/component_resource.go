package statuspage

import (
	"context"
	"strings"

	"terraform-provider-openstatus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*componentResource)(nil)
	_ resource.ResourceWithConfigure   = (*componentResource)(nil)
	_ resource.ResourceWithImportState = (*componentResource)(nil)
)

func NewComponentResource() resource.Resource {
	return &componentResource{}
}

type componentResource struct {
	client *client.Client
}

type componentModel struct {
	ID          types.String `tfsdk:"id"`
	PageID      types.String `tfsdk:"page_id"`
	Type        types.String `tfsdk:"type"`
	MonitorID   types.String `tfsdk:"monitor_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Order       types.Int64  `tfsdk:"order"`
	GroupID     types.String `tfsdk:"group_id"`
	GroupOrder  types.Int64  `tfsdk:"group_order"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *componentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_component"
}

func (r *componentResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

func (r *componentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a status page component (monitor or static).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"page_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.OneOf("monitor", "static")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"monitor_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Required when type is `monitor`.",
			},
			"name": schema.StringAttribute{
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtMost(256)},
			},
			"description": schema.StringAttribute{
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtMost(1024)},
			},
			"order": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Display order of the component's group on the status page.",
			},
			"group_id": schema.StringAttribute{
				Optional: true,
			},
			"group_order": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Display order of the component within its group.",
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

type apiAddMonitorComponentRequest struct {
	PageID      string `json:"pageId"`
	MonitorID   string `json:"monitorId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Order       *int64 `json:"order,omitempty"`
	GroupID     string `json:"groupId,omitempty"`
	GroupOrder  *int64 `json:"groupOrder,omitempty"`
}

type apiAddStaticComponentRequest struct {
	PageID      string `json:"pageId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Order       *int64 `json:"order,omitempty"`
	GroupID     string `json:"groupId,omitempty"`
	GroupOrder  *int64 `json:"groupOrder,omitempty"`
}

type apiComponentResponse struct {
	Component apiComponent `json:"component"`
}

type apiComponent struct {
	ID          string `json:"id"`
	PageID      string `json:"pageId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	MonitorID   string `json:"monitorId"`
	Order       *int64 `json:"order"`
	GroupID     string `json:"groupId"`
	GroupOrder  *int64 `json:"groupOrder"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type apiUpdateComponentRequest struct {
	ID          string  `json:"id"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Order       *int64  `json:"order,omitempty"`
	GroupID     *string `json:"groupId,omitempty"`
	GroupOrder  *int64  `json:"groupOrder,omitempty"`
}

type apiStatusPageContentResponse struct {
	Components []apiComponent      `json:"components"`
	Groups     []apiComponentGroup `json:"groups"`
}

func (r *componentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data componentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp apiComponentResponse
	var err error

	if data.Type.ValueString() == "monitor" {
		apiReq := apiAddMonitorComponentRequest{
			PageID:      data.PageID.ValueString(),
			MonitorID:   data.MonitorID.ValueString(),
			Name:        data.Name.ValueString(),
			Description: data.Description.ValueString(),
			GroupID:     data.GroupID.ValueString(),
		}
		if !data.Order.IsNull() && !data.Order.IsUnknown() {
			v := data.Order.ValueInt64()
			apiReq.Order = &v
		}
		if !data.GroupOrder.IsNull() && !data.GroupOrder.IsUnknown() {
			v := data.GroupOrder.ValueInt64()
			apiReq.GroupOrder = &v
		}
		err = r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/AddMonitorComponent", apiReq, &apiResp)
	} else {
		apiReq := apiAddStaticComponentRequest{
			PageID:      data.PageID.ValueString(),
			Name:        data.Name.ValueString(),
			Description: data.Description.ValueString(),
			GroupID:     data.GroupID.ValueString(),
		}
		if !data.Order.IsNull() && !data.Order.IsUnknown() {
			v := data.Order.ValueInt64()
			apiReq.Order = &v
		}
		if !data.GroupOrder.IsNull() && !data.GroupOrder.IsUnknown() {
			v := data.GroupOrder.ValueInt64()
			apiReq.GroupOrder = &v
		}
		err = r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/AddStaticComponent", apiReq, &apiResp)
	}

	if err != nil {
		resp.Diagnostics.AddError("Error creating component", err.Error())
		return
	}

	componentAPIToModel(apiResp.Component, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *componentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data componentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	comp, err := r.findComponent(ctx, data.PageID.ValueString(), data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading component", err.Error())
		return
	}
	if comp == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	componentAPIToModel(*comp, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *componentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data componentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state componentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := apiUpdateComponentRequest{ID: state.ID.ValueString()}
	if !data.Name.IsNull() {
		v := data.Name.ValueString()
		updateReq.Name = &v
	}
	if !data.Description.IsNull() {
		v := data.Description.ValueString()
		updateReq.Description = &v
	}
	if !data.Order.IsNull() && !data.Order.IsUnknown() {
		v := data.Order.ValueInt64()
		updateReq.Order = &v
	}
	if !data.GroupID.IsNull() && !data.GroupID.IsUnknown() {
		v := data.GroupID.ValueString()
		updateReq.GroupID = &v
	}
	if !data.GroupOrder.IsNull() && !data.GroupOrder.IsUnknown() {
		v := data.GroupOrder.ValueInt64()
		updateReq.GroupOrder = &v
	}

	var apiResp apiComponentResponse
	err := r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/UpdateComponent", updateReq, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error updating component", err.Error())
		return
	}

	componentAPIToModel(apiResp.Component, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *componentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data componentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/RemoveComponent",
		map[string]string{"id": data.ID.ValueString()}, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting component", err.Error())
	}
}

func (r *componentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: page_id/component_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("page_id"), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(parts[1]))...)
}

func (r *componentResource) findComponent(ctx context.Context, pageID, componentID string) (*apiComponent, error) {
	var contentResp apiStatusPageContentResponse
	err := r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/GetStatusPageContent",
		map[string]string{"id": pageID}, &contentResp)
	if err != nil {
		return nil, err
	}
	for _, c := range contentResp.Components {
		if c.ID == componentID {
			return &c, nil
		}
	}
	return nil, nil
}

func componentAPIToModel(api apiComponent, data *componentModel) {
	data.ID = types.StringValue(api.ID)
	data.PageID = types.StringValue(api.PageID)
	if api.Name != "" {
		data.Name = types.StringValue(api.Name)
	}
	if api.Description != "" {
		data.Description = types.StringValue(api.Description)
	}
	if api.Order != nil {
		data.Order = types.Int64Value(*api.Order)
	}
	if api.GroupOrder != nil {
		data.GroupOrder = types.Int64Value(*api.GroupOrder)
	}
	if api.GroupID != "" {
		data.GroupID = types.StringValue(api.GroupID)
	}
	if api.MonitorID != "" {
		data.MonitorID = types.StringValue(api.MonitorID)
	}

	switch api.Type {
	case "PAGE_COMPONENT_TYPE_MONITOR":
		data.Type = types.StringValue("monitor")
	case "PAGE_COMPONENT_TYPE_STATIC":
		data.Type = types.StringValue("static")
	default:
		data.Type = types.StringValue(api.Type)
	}

	data.CreatedAt = types.StringValue(api.CreatedAt)
	data.UpdatedAt = types.StringValue(api.UpdatedAt)
}

