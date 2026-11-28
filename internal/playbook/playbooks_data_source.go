package playbook

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	playbookv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/playbook/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*playbooksDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*playbooksDataSource)(nil)
)

func NewPlaybooksDataSource() datasource.DataSource {
	return &playbooksDataSource{}
}

type playbooksDataSource struct {
	client *client.Client
}

type playbooksDataSourceModel struct {
	Playbooks types.List `tfsdk:"playbooks"`
}

var playbookSummaryObjTypes = map[string]attr.Type{
	"slug":        types.StringType,
	"title":       types.StringType,
	"description": types.StringType,
	"created_at":  types.StringType,
	"updated_at":  types.StringType,
}

func (d *playbooksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbooks"
}

func (d *playbooksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(providerConfig).Client
}

func (d *playbooksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the playbooks the workspace has written: built-in slugs first, then custom ones by age.\n\n" +
			"Content is not included; use the `openstatus_playbook` data source to fetch one by slug.",
		Attributes: map[string]schema.Attribute{
			"playbooks": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"slug":        schema.StringAttribute{Computed: true},
						"title":       schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"created_at":  schema.StringAttribute{Computed: true},
						"updated_at":  schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *playbooksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data playbooksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := d.client.Playbook.ListPlaybooks(ctx, connect.NewRequest(&playbookv1.ListPlaybooksRequest{}))
	if err != nil {
		resp.Diagnostics.AddError("Error listing playbooks", err.Error())
		return
	}

	objType := types.ObjectType{AttrTypes: playbookSummaryObjTypes}
	items := make([]attr.Value, 0, len(apiResp.Msg.GetPlaybooks()))
	for _, p := range apiResp.Msg.GetPlaybooks() {
		obj, diags := types.ObjectValue(playbookSummaryObjTypes, map[string]attr.Value{
			"slug":        types.StringValue(p.GetSlug()),
			"title":       types.StringValue(p.GetTitle()),
			"description": types.StringValue(p.GetDescription()),
			"created_at":  types.StringValue(p.GetCreatedAt()),
			"updated_at":  types.StringValue(p.GetUpdatedAt()),
		})
		resp.Diagnostics.Append(diags...)
		items = append(items, obj)
	}

	list, diags := types.ListValue(objType, items)
	resp.Diagnostics.Append(diags...)
	data.Playbooks = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
