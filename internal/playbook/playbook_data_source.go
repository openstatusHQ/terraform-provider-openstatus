package playbook

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	playbookv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/playbook/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*playbookDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*playbookDataSource)(nil)
)

func NewPlaybookDataSource() datasource.DataSource {
	return &playbookDataSource{}
}

type playbookDataSource struct {
	client *client.Client
}

type playbookDataSourceModel struct {
	Slug        types.String `tfsdk:"slug"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Content     types.String `tfsdk:"content"`
	Variables   types.Map    `tfsdk:"variables"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *playbookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook"
}

func (d *playbookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(providerConfig).Client
}

func (d *playbookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a playbook the workspace has written, including its content. " +
			"A built-in slug the workspace has never written is not found.",
		Attributes: map[string]schema.Attribute{
			"slug": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Slug of the playbook.",
			},
			"title":       schema.StringAttribute{Computed: true},
			"description": schema.StringAttribute{Computed: true},
			"content": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The playbook in markdown, as written. `{{name}}` marks a variable.",
			},
			"variables": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Values set for the variables in the content, by name.",
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *playbookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data playbookDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &playbookv1.GetPlaybookRequest{}
	getReq.SetSlug(data.Slug.ValueString())

	apiResp, err := d.client.Playbook.GetPlaybook(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Error reading playbook", err.Error())
		return
	}

	api := apiResp.Msg.GetPlaybook()
	data.Slug = types.StringValue(api.GetSlug())
	data.Title = types.StringValue(api.GetTitle())
	data.Description = types.StringValue(api.GetDescription())
	data.Content = types.StringValue(api.GetContent())
	data.CreatedAt = types.StringValue(api.GetCreatedAt())
	data.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	variables, diags := types.MapValueFrom(ctx, types.StringType, api.GetVariables())
	resp.Diagnostics.Append(diags...)
	data.Variables = variables

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
