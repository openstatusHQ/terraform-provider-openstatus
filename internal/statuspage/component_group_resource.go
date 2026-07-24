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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*componentGroupResource)(nil)
	_ resource.ResourceWithConfigure   = (*componentGroupResource)(nil)
	_ resource.ResourceWithImportState = (*componentGroupResource)(nil)
)

func NewComponentGroupResource() resource.Resource {
	return &componentGroupResource{}
}

type componentGroupResource struct {
	client *client.Client
}

type componentGroupModel struct {
	ID          types.String `tfsdk:"id"`
	PageID      types.String `tfsdk:"page_id"`
	Name        types.String `tfsdk:"name"`
	DefaultOpen types.Bool   `tfsdk:"default_open"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *componentGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page_component_group"
}

func (r *componentGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

func (r *componentGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a status page component group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"page_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 256)},
			},
			"default_open": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the group should be expanded by default on the status page.",
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *componentGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data componentGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &statuspagev1.CreateComponentGroupRequest{}
	apiReq.SetPageId(data.PageID.ValueString())
	apiReq.SetName(data.Name.ValueString())
	apiReq.SetDefaultOpen(data.DefaultOpen.ValueBool())

	apiResp, err := r.client.StatusPage.CreateComponentGroup(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error creating component group", err.Error())
		return
	}

	componentGroupAPIToModel(apiResp.Msg.GetGroup(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *componentGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data componentGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.findGroup(ctx, data.PageID.ValueString(), data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading component group", err.Error())
		return
	}
	if group == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	componentGroupAPIToModel(group, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *componentGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data componentGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state componentGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &statuspagev1.UpdateComponentGroupRequest{}
	updateReq.SetId(state.ID.ValueString())
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetDefaultOpen(data.DefaultOpen.ValueBool())

	apiResp, err := r.client.StatusPage.UpdateComponentGroup(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating component group", err.Error())
		return
	}

	componentGroupAPIToModel(apiResp.Msg.GetGroup(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *componentGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data componentGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &statuspagev1.DeleteComponentGroupRequest{}
	deleteReq.SetId(data.ID.ValueString())

	_, err := r.client.StatusPage.DeleteComponentGroup(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting component group", err.Error())
	}
}

func (r *componentGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: page_id/group_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("page_id"), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(parts[1]))...)
}

func (r *componentGroupResource) findGroup(ctx context.Context, pageID, groupID string) (*statuspagev1.PageComponentGroup, error) {
	contentReq := &statuspagev1.GetStatusPageContentRequest{}
	contentReq.SetId(pageID)

	contentResp, err := r.client.StatusPage.GetStatusPageContent(ctx, connect.NewRequest(contentReq))
	if err != nil {
		return nil, err
	}
	for _, g := range contentResp.Msg.GetGroups() {
		if g.GetId() == groupID {
			return g, nil
		}
	}
	return nil, nil
}

func componentGroupAPIToModel(api *statuspagev1.PageComponentGroup, data *componentGroupModel) {
	data.ID = types.StringValue(api.GetId())
	data.PageID = types.StringValue(api.GetPageId())
	data.Name = types.StringValue(api.GetName())
	data.DefaultOpen = types.BoolValue(api.GetDefaultOpen())
	data.CreatedAt = types.StringValue(api.GetCreatedAt())
	data.UpdatedAt = types.StringValue(api.GetUpdatedAt())
}
