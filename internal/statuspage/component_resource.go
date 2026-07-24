package statuspage

import (
	"context"
	"strings"

	"terraform-provider-openstatus/internal/client"

	statuspagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"

	"connectrpc.com/connect"

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

func (r *componentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data componentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var component *statuspagev1.PageComponent
	var err error

	if data.Type.ValueString() == "monitor" {
		apiReq := &statuspagev1.AddMonitorComponentRequest{}
		apiReq.SetPageId(data.PageID.ValueString())
		apiReq.SetMonitorId(data.MonitorID.ValueString())
		apiReq.SetName(data.Name.ValueString())
		apiReq.SetDescription(data.Description.ValueString())
		apiReq.SetGroupId(data.GroupID.ValueString())
		if !data.Order.IsNull() && !data.Order.IsUnknown() {
			apiReq.SetOrder(int32(data.Order.ValueInt64()))
		}
		var apiResp *connect.Response[statuspagev1.AddMonitorComponentResponse]
		apiResp, err = r.client.StatusPage.AddMonitorComponent(ctx, connect.NewRequest(apiReq))
		if apiResp != nil {
			component = apiResp.Msg.GetComponent()
		}
	} else {
		apiReq := &statuspagev1.AddStaticComponentRequest{}
		apiReq.SetPageId(data.PageID.ValueString())
		apiReq.SetName(data.Name.ValueString())
		apiReq.SetDescription(data.Description.ValueString())
		apiReq.SetGroupId(data.GroupID.ValueString())
		if !data.Order.IsNull() && !data.Order.IsUnknown() {
			apiReq.SetOrder(int32(data.Order.ValueInt64()))
		}
		var apiResp *connect.Response[statuspagev1.AddStaticComponentResponse]
		apiResp, err = r.client.StatusPage.AddStaticComponent(ctx, connect.NewRequest(apiReq))
		if apiResp != nil {
			component = apiResp.Msg.GetComponent()
		}
	}

	if err != nil {
		resp.Diagnostics.AddError("Error creating component", err.Error())
		return
	}

	componentAPIToModel(component, &data)
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
		if client.IsNotFound(err) {
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

	componentAPIToModel(comp, &data)
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

	updateReq := &statuspagev1.UpdateComponentRequest{}
	updateReq.SetId(state.ID.ValueString())
	if !data.Name.IsNull() {
		updateReq.SetName(data.Name.ValueString())
	}
	if !data.Description.IsNull() {
		updateReq.SetDescription(data.Description.ValueString())
	}
	if !data.Order.IsNull() && !data.Order.IsUnknown() {
		updateReq.SetOrder(int32(data.Order.ValueInt64()))
	}
	if !data.GroupID.IsNull() && !data.GroupID.IsUnknown() {
		updateReq.SetGroupId(data.GroupID.ValueString())
	}
	if !data.GroupOrder.IsNull() && !data.GroupOrder.IsUnknown() {
		updateReq.SetGroupOrder(int32(data.GroupOrder.ValueInt64()))
	}

	apiResp, err := r.client.StatusPage.UpdateComponent(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating component", err.Error())
		return
	}

	componentAPIToModel(apiResp.Msg.GetComponent(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *componentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data componentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	removeReq := &statuspagev1.RemoveComponentRequest{}
	removeReq.SetId(data.ID.ValueString())

	_, err := r.client.StatusPage.RemoveComponent(ctx, connect.NewRequest(removeReq))
	if err != nil && !client.IsNotFound(err) {
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

func (r *componentResource) findComponent(ctx context.Context, pageID, componentID string) (*statuspagev1.PageComponent, error) {
	contentReq := &statuspagev1.GetStatusPageContentRequest{}
	contentReq.SetId(pageID)

	contentResp, err := r.client.StatusPage.GetStatusPageContent(ctx, connect.NewRequest(contentReq))
	if err != nil {
		return nil, err
	}
	for _, c := range contentResp.Msg.GetComponents() {
		if c.GetId() == componentID {
			return c, nil
		}
	}
	return nil, nil
}

// order and group_order carry no field presence in the proto, and protojson
// omits zero values, so a zero from the API is indistinguishable from an
// omitted field. Both are treated as "not reported": keep the existing value,
// collapsing unknown to null.
func componentAPIToModel(api *statuspagev1.PageComponent, data *componentModel) {
	data.ID = types.StringValue(api.GetId())
	data.PageID = types.StringValue(api.GetPageId())
	if api.GetName() != "" {
		data.Name = types.StringValue(api.GetName())
	}
	if api.GetDescription() != "" {
		data.Description = types.StringValue(api.GetDescription())
	}
	if api.GetOrder() != 0 {
		data.Order = types.Int64Value(int64(api.GetOrder()))
	} else if data.Order.IsUnknown() {
		data.Order = types.Int64Null()
	}
	if api.GetGroupOrder() != 0 {
		data.GroupOrder = types.Int64Value(int64(api.GetGroupOrder()))
	} else if data.GroupOrder.IsUnknown() {
		data.GroupOrder = types.Int64Null()
	}
	if api.GetGroupId() != "" {
		data.GroupID = types.StringValue(api.GetGroupId())
	}
	if api.GetMonitorId() != "" {
		data.MonitorID = types.StringValue(api.GetMonitorId())
	}

	switch api.GetType() {
	case statuspagev1.PageComponentType_PAGE_COMPONENT_TYPE_MONITOR:
		data.Type = types.StringValue("monitor")
	case statuspagev1.PageComponentType_PAGE_COMPONENT_TYPE_STATIC:
		data.Type = types.StringValue("static")
	default:
		data.Type = types.StringValue(api.GetType().String())
	}

	data.CreatedAt = types.StringValue(api.GetCreatedAt())
	data.UpdatedAt = types.StringValue(api.GetUpdatedAt())
}
