package notification

import (
	"context"
	"fmt"

	"terraform-provider-openstatus/internal/client"

	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*notificationResource)(nil)
	_ resource.ResourceWithConfigure   = (*notificationResource)(nil)
	_ resource.ResourceWithImportState = (*notificationResource)(nil)
)

type providerConfig = client.ProviderConfig

func NewNotificationResource() resource.Resource {
	return &notificationResource{}
}

type notificationResource struct {
	client *client.Client
}

type notificationModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ProviderType types.String `tfsdk:"provider_type"`
	MonitorIDs   types.Set    `tfsdk:"monitor_ids"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`

	Discord       types.List `tfsdk:"discord"`
	Email         types.List `tfsdk:"email"`
	Slack         types.List `tfsdk:"slack"`
	PagerDuty     types.List `tfsdk:"pagerduty"`
	Opsgenie      types.List `tfsdk:"opsgenie"`
	Webhook       types.List `tfsdk:"webhook"`
	Telegram      types.List `tfsdk:"telegram"`
	SMS           types.List `tfsdk:"sms"`
	WhatsApp      types.List `tfsdk:"whatsapp"`
	GoogleChat    types.List `tfsdk:"google_chat"`
	GrafanaOncall types.List `tfsdk:"grafana_oncall"`
	Ntfy          types.List `tfsdk:"ntfy"`
	MSTeams       types.List `tfsdk:"ms_teams"`
}

func (r *notificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification"
}

func (r *notificationResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

var webhookHeaderObjTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
}

func (r *notificationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	providerTypes := []string{
		"discord", "email", "slack", "pagerduty", "opsgenie", "webhook",
		"telegram", "sms", "whatsapp", "google_chat", "grafana_oncall", "ntfy",
		"ms_teams",
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a notification channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"provider_type": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.OneOf(providerTypes...)},
			},
			"monitor_ids": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
		},
		Blocks: map[string]schema.Block{
			"discord": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"webhook_url": schema.StringAttribute{Required: true, Sensitive: true},
					},
				},
			},
			"email": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"email": schema.StringAttribute{Required: true},
					},
				},
			},
			"slack": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"webhook_url": schema.StringAttribute{Required: true, Sensitive: true},
					},
				},
			},
			"pagerduty": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"integration_key": schema.StringAttribute{Required: true, Sensitive: true},
					},
				},
			},
			"opsgenie": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"api_key": schema.StringAttribute{Required: true, Sensitive: true},
						"region": schema.StringAttribute{
							Required:   true,
							Validators: []validator.String{stringvalidator.OneOf("us", "eu")},
						},
					},
				},
			},
			"webhook": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"endpoint": schema.StringAttribute{Required: true},
						"headers": schema.ListNestedAttribute{
							Optional: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"key":   schema.StringAttribute{Required: true},
									"value": schema.StringAttribute{Required: true},
								},
							},
						},
					},
				},
			},
			"telegram": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"chat_id": schema.StringAttribute{Required: true},
					},
				},
			},
			"sms": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"phone_number": schema.StringAttribute{Required: true},
					},
				},
			},
			"whatsapp": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"phone_number": schema.StringAttribute{Required: true},
					},
				},
			},
			"google_chat": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"webhook_url": schema.StringAttribute{Required: true, Sensitive: true},
					},
				},
			},
			"grafana_oncall": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"webhook_url": schema.StringAttribute{Required: true, Sensitive: true},
					},
				},
			},
			"ntfy": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"topic":      schema.StringAttribute{Required: true},
						"server_url": schema.StringAttribute{Optional: true},
						"token":      schema.StringAttribute{Optional: true, Sensitive: true},
					},
				},
			},
			"ms_teams": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"webhook_url": schema.StringAttribute{Required: true, Sensitive: true},
					},
				},
			},
		},
	}
}

var providerTypeToAPI = map[string]notificationv1.NotificationProvider{
	"discord":        notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_DISCORD,
	"email":          notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_EMAIL,
	"slack":          notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_SLACK,
	"pagerduty":      notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_PAGERDUTY,
	"opsgenie":       notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_OPSGENIE,
	"webhook":        notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_WEBHOOK,
	"telegram":       notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_TELEGRAM,
	"sms":            notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_SMS,
	"whatsapp":       notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_WHATSAPP,
	"google_chat":    notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_GOOGLE_CHAT,
	"grafana_oncall": notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_GRAFANA_ONCALL,
	"ntfy":           notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_NTFY,
	"ms_teams":       notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_MS_TEAMS,
}

var providerTypeFromAPI = func() map[notificationv1.NotificationProvider]string {
	m := make(map[notificationv1.NotificationProvider]string, len(providerTypeToAPI))
	for k, v := range providerTypeToAPI {
		m[v] = k
	}
	return m
}()

var opsgenieRegionToAPI = map[string]notificationv1.OpsgenieRegion{
	"us": notificationv1.OpsgenieRegion_OPSGENIE_REGION_US,
	"eu": notificationv1.OpsgenieRegion_OPSGENIE_REGION_EU,
}

var opsgenieRegionFromAPI = func() map[notificationv1.OpsgenieRegion]string {
	m := make(map[notificationv1.OpsgenieRegion]string, len(opsgenieRegionToAPI))
	for k, v := range opsgenieRegionToAPI {
		m[v] = k
	}
	return m
}()

func (r *notificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data notificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, diags := notificationModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.Notification.CreateNotification(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error creating notification", err.Error())
		return
	}

	resp.Diagnostics.Append(notificationAPIToModel(ctx, apiResp.Msg.GetNotification(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *notificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data notificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &notificationv1.GetNotificationRequest{}
	getReq.SetId(data.ID.ValueString())

	apiResp, err := r.client.Notification.GetNotification(ctx, connect.NewRequest(getReq))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading notification", err.Error())
		return
	}

	resp.Diagnostics.Append(notificationAPIToModel(ctx, apiResp.Msg.GetNotification(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *notificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data notificationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state notificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerData, diags := extractProviderData(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	monitorIDs := []string{}
	if !data.MonitorIDs.IsNull() && !data.MonitorIDs.IsUnknown() {
		resp.Diagnostics.Append(data.MonitorIDs.ElementsAs(ctx, &monitorIDs, false)...)
	}

	updateReq := newNotificationUpdateRequest(state.ID.ValueString(), data.Name.ValueString(), providerData, monitorIDs)

	apiResp, err := r.client.Notification.UpdateNotification(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating notification", err.Error())
		return
	}

	resp.Diagnostics.Append(notificationAPIToModel(ctx, apiResp.Msg.GetNotification(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *notificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data notificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &notificationv1.DeleteNotificationRequest{}
	deleteReq.SetId(data.ID.ValueString())

	_, err := r.client.Notification.DeleteNotification(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting notification", err.Error())
	}
}

func (r *notificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
}

func notificationModelToAPI(ctx context.Context, data notificationModel) (*notificationv1.CreateNotificationRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	providerAPI, ok := providerTypeToAPI[data.ProviderType.ValueString()]
	if !ok {
		diags.AddError("Invalid provider type", "Unknown provider: "+data.ProviderType.ValueString())
		return nil, diags
	}

	providerData, d := extractProviderData(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	var monitorIDs []string
	if !data.MonitorIDs.IsNull() && !data.MonitorIDs.IsUnknown() {
		diags.Append(data.MonitorIDs.ElementsAs(ctx, &monitorIDs, false)...)
	}

	req := &notificationv1.CreateNotificationRequest{}
	req.SetName(data.Name.ValueString())
	req.SetProvider(providerAPI)
	req.SetData(providerData)
	req.SetMonitorIds(monitorIDs)
	return req, diags
}

// newNotificationUpdateRequest always sets update_monitor_ids so Terraform
// stays authoritative over the association.
func newNotificationUpdateRequest(id, name string, data *notificationv1.NotificationData, monitorIDs []string) *notificationv1.UpdateNotificationRequest {
	req := &notificationv1.UpdateNotificationRequest{}
	req.SetId(id)
	req.SetName(name)
	req.SetData(data)
	req.SetMonitorIds(monitorIDs)
	req.SetUpdateMonitorIds(true)
	return req
}

func singleBlock(list types.List, diags *diag.Diagnostics, label string) (types.Object, bool) {
	elems := list.Elements()
	if len(elems) == 0 {
		diags.AddError("Missing block", "Expected exactly one "+label+" block")
		return types.Object{}, false
	}
	obj, ok := elems[0].(types.Object)
	if !ok {
		diags.AddError("Invalid block", "Expected an object for the "+label+" block")
		return types.Object{}, false
	}
	return obj, true
}

func stringAttr(obj types.Object, name string) string {
	v, ok := obj.Attributes()[name]
	if !ok {
		return ""
	}
	s, ok := v.(types.String)
	if !ok {
		return ""
	}
	return s.ValueString()
}

func extractProviderData(ctx context.Context, data notificationModel) (*notificationv1.NotificationData, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := &notificationv1.NotificationData{}
	pt := data.ProviderType.ValueString()

	webhookURL := func(list types.List, label string) string {
		obj, ok := singleBlock(list, &diags, label)
		if !ok {
			return ""
		}
		return stringAttr(obj, "webhook_url")
	}

	switch pt {
	case "discord":
		inner := &notificationv1.DiscordData{}
		inner.SetWebhookUrl(webhookURL(data.Discord, "discord"))
		result.SetDiscord(inner)
	case "email":
		obj, ok := singleBlock(data.Email, &diags, "email")
		if !ok {
			return nil, diags
		}
		inner := &notificationv1.EmailData{}
		inner.SetEmail(stringAttr(obj, "email"))
		result.SetEmail(inner)
	case "slack":
		inner := &notificationv1.SlackData{}
		inner.SetWebhookUrl(webhookURL(data.Slack, "slack"))
		result.SetSlack(inner)
	case "pagerduty":
		obj, ok := singleBlock(data.PagerDuty, &diags, "pagerduty")
		if !ok {
			return nil, diags
		}
		inner := &notificationv1.PagerDutyData{}
		inner.SetIntegrationKey(stringAttr(obj, "integration_key"))
		result.SetPagerduty(inner)
	case "opsgenie":
		obj, ok := singleBlock(data.Opsgenie, &diags, "opsgenie")
		if !ok {
			return nil, diags
		}
		inner := &notificationv1.OpsgenieData{}
		inner.SetApiKey(stringAttr(obj, "api_key"))
		inner.SetRegion(opsgenieRegionToAPI[stringAttr(obj, "region")])
		result.SetOpsgenie(inner)
	case "webhook":
		inner, d := extractWebhookEndpointBlock(ctx, data.Webhook)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		result.SetWebhook(inner)
	case "telegram":
		obj, ok := singleBlock(data.Telegram, &diags, "telegram")
		if !ok {
			return nil, diags
		}
		inner := &notificationv1.TelegramData{}
		inner.SetChatId(stringAttr(obj, "chat_id"))
		result.SetTelegram(inner)
	case "sms":
		obj, ok := singleBlock(data.SMS, &diags, "sms")
		if !ok {
			return nil, diags
		}
		inner := &notificationv1.SmsData{}
		inner.SetPhoneNumber(stringAttr(obj, "phone_number"))
		result.SetSms(inner)
	case "whatsapp":
		obj, ok := singleBlock(data.WhatsApp, &diags, "whatsapp")
		if !ok {
			return nil, diags
		}
		inner := &notificationv1.WhatsappData{}
		inner.SetPhoneNumber(stringAttr(obj, "phone_number"))
		result.SetWhatsapp(inner)
	case "google_chat":
		inner := &notificationv1.GoogleChatData{}
		inner.SetWebhookUrl(webhookURL(data.GoogleChat, "google_chat"))
		result.SetGoogleChat(inner)
	case "grafana_oncall":
		inner := &notificationv1.GrafanaOncallData{}
		inner.SetWebhookUrl(webhookURL(data.GrafanaOncall, "grafana_oncall"))
		result.SetGrafanaOncall(inner)
	case "ntfy":
		inner, d := extractNtfyBlock(data.Ntfy)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		result.SetNtfy(inner)
	case "ms_teams":
		inner := &notificationv1.MsTeamsData{}
		inner.SetWebhookUrl(webhookURL(data.MSTeams, "ms_teams"))
		result.SetMsTeams(inner)
	default:
		diags.AddError("Unknown provider", "No data block for provider: "+pt)
		return nil, diags
	}

	if diags.HasError() {
		return nil, diags
	}
	return result, diags
}

func extractWebhookEndpointBlock(ctx context.Context, list types.List) (*notificationv1.WebhookData, diag.Diagnostics) {
	var diags diag.Diagnostics
	var items []struct {
		Endpoint string `tfsdk:"endpoint"`
		Headers  []struct {
			Key   string `tfsdk:"key"`
			Value string `tfsdk:"value"`
		} `tfsdk:"headers"`
	}
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	if len(items) == 0 {
		diags.AddError("Missing block", "Expected exactly one webhook block")
		return nil, diags
	}

	result := &notificationv1.WebhookData{}
	result.SetEndpoint(items[0].Endpoint)
	if len(items[0].Headers) > 0 {
		headers := make([]*notificationv1.WebhookHeader, 0, len(items[0].Headers))
		for _, h := range items[0].Headers {
			header := &notificationv1.WebhookHeader{}
			header.SetKey(h.Key)
			header.SetValue(h.Value)
			headers = append(headers, header)
		}
		result.SetHeaders(headers)
	}
	return result, diags
}

// extractNtfyBlock only sets the optional fields when non-empty, so an omitted
// server_url or token is absent on the wire rather than an empty string.
func extractNtfyBlock(list types.List) (*notificationv1.NtfyData, diag.Diagnostics) {
	var diags diag.Diagnostics
	obj, ok := singleBlock(list, &diags, "ntfy")
	if !ok {
		return nil, diags
	}

	result := &notificationv1.NtfyData{}
	result.SetTopic(stringAttr(obj, "topic"))
	if v := stringAttr(obj, "server_url"); v != "" {
		result.SetServerUrl(v)
	}
	if v := stringAttr(obj, "token"); v != "" {
		result.SetToken(v)
	}
	return result, diags
}

func notificationAPIToModel(ctx context.Context, api *notificationv1.Notification, data *notificationModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(api.GetId())
	data.Name = types.StringValue(api.GetName())
	data.CreatedAt = types.StringValue(api.GetCreatedAt())
	data.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	if pt, ok := providerTypeFromAPI[api.GetProvider()]; ok {
		data.ProviderType = types.StringValue(pt)
	} else {
		diags.AddWarning(
			"Unknown notification provider",
			fmt.Sprintf(
				"OpenStatus returned provider %q which this provider version does not recognize. "+
					"State for notification %q may be incomplete; upgrade the openstatus provider to manage this resource.",
				api.GetProvider().String(), api.GetId(),
			),
		)
	}

	if len(api.GetMonitorIds()) > 0 {
		monitorSet, d := types.SetValueFrom(ctx, types.StringType, api.GetMonitorIds())
		diags.Append(d...)
		data.MonitorIDs = monitorSet
	} else {
		data.MonitorIDs = types.SetNull(types.StringType)
	}

	pt := data.ProviderType.ValueString()
	apiData := api.GetData()

	nullList := func(attrTypes map[string]attr.Type) types.List {
		return types.ListNull(types.ObjectType{AttrTypes: attrTypes})
	}

	discordType := map[string]attr.Type{"webhook_url": types.StringType}
	emailType := map[string]attr.Type{"email": types.StringType}
	slackType := map[string]attr.Type{"webhook_url": types.StringType}
	pagerdutyType := map[string]attr.Type{"integration_key": types.StringType}
	opsgenieType := map[string]attr.Type{"api_key": types.StringType, "region": types.StringType}
	webhookType := map[string]attr.Type{
		"endpoint": types.StringType,
		"headers":  types.ListType{ElemType: types.ObjectType{AttrTypes: webhookHeaderObjTypes}},
	}
	telegramType := map[string]attr.Type{"chat_id": types.StringType}
	smsType := map[string]attr.Type{"phone_number": types.StringType}
	whatsappType := map[string]attr.Type{"phone_number": types.StringType}
	googleChatType := map[string]attr.Type{"webhook_url": types.StringType}
	grafanaOncallType := map[string]attr.Type{"webhook_url": types.StringType}
	ntfyType := map[string]attr.Type{"topic": types.StringType, "server_url": types.StringType, "token": types.StringType}
	msTeamsType := map[string]attr.Type{"webhook_url": types.StringType}

	// Capture sensitive values that the API does not echo back before the
	// per-provider lists are reset below. See ntfy.token note in the switch.
	plannedNtfyToken := existingNtfyStringField(data.Ntfy, "token")

	data.Discord = nullList(discordType)
	data.Email = nullList(emailType)
	data.Slack = nullList(slackType)
	data.PagerDuty = nullList(pagerdutyType)
	data.Opsgenie = nullList(opsgenieType)
	data.Webhook = nullList(webhookType)
	data.Telegram = nullList(telegramType)
	data.SMS = nullList(smsType)
	data.WhatsApp = nullList(whatsappType)
	data.GoogleChat = nullList(googleChatType)
	data.GrafanaOncall = nullList(grafanaOncallType)
	data.Ntfy = nullList(ntfyType)
	data.MSTeams = nullList(msTeamsType)

	switch pt {
	case "discord":
		data.Discord = buildSingleObjList(ctx, discordType, map[string]attr.Value{
			"webhook_url": types.StringValue(apiData.GetDiscord().GetWebhookUrl()),
		}, &diags)
	case "email":
		data.Email = buildSingleObjList(ctx, emailType, map[string]attr.Value{
			"email": types.StringValue(apiData.GetEmail().GetEmail()),
		}, &diags)
	case "slack":
		data.Slack = buildSingleObjList(ctx, slackType, map[string]attr.Value{
			"webhook_url": types.StringValue(apiData.GetSlack().GetWebhookUrl()),
		}, &diags)
	case "pagerduty":
		data.PagerDuty = buildSingleObjList(ctx, pagerdutyType, map[string]attr.Value{
			"integration_key": types.StringValue(apiData.GetPagerduty().GetIntegrationKey()),
		}, &diags)
	case "opsgenie":
		apiRegion := apiData.GetOpsgenie().GetRegion()
		region := ""
		if r, ok := opsgenieRegionFromAPI[apiRegion]; ok {
			region = r
		} else if apiRegion != notificationv1.OpsgenieRegion_OPSGENIE_REGION_UNSPECIFIED {
			region = apiRegion.String()
			diags.AddWarning(
				"Unknown Opsgenie region",
				fmt.Sprintf(
					"OpenStatus returned Opsgenie region %q which this provider version does not recognize. "+
						"State for notification %q may show the raw enum value.",
					region, api.GetId(),
				),
			)
		}
		data.Opsgenie = buildSingleObjList(ctx, opsgenieType, map[string]attr.Value{
			"api_key": types.StringValue(apiData.GetOpsgenie().GetApiKey()),
			"region":  types.StringValue(region),
		}, &diags)
	case "webhook":
		headersVal := types.ListNull(types.ObjectType{AttrTypes: webhookHeaderObjTypes})
		if apiHeaders := apiData.GetWebhook().GetHeaders(); len(apiHeaders) > 0 {
			headerObjs := make([]attr.Value, 0, len(apiHeaders))
			for _, h := range apiHeaders {
				obj, d := types.ObjectValue(webhookHeaderObjTypes, map[string]attr.Value{
					"key":   types.StringValue(h.GetKey()),
					"value": types.StringValue(h.GetValue()),
				})
				diags.Append(d...)
				headerObjs = append(headerObjs, obj)
			}
			headersVal, _ = types.ListValue(types.ObjectType{AttrTypes: webhookHeaderObjTypes}, headerObjs)
		}
		data.Webhook = buildSingleObjList(ctx, webhookType, map[string]attr.Value{
			"endpoint": types.StringValue(apiData.GetWebhook().GetEndpoint()),
			"headers":  headersVal,
		}, &diags)
	case "telegram":
		data.Telegram = buildSingleObjList(ctx, telegramType, map[string]attr.Value{
			"chat_id": types.StringValue(apiData.GetTelegram().GetChatId()),
		}, &diags)
	case "sms":
		data.SMS = buildSingleObjList(ctx, smsType, map[string]attr.Value{
			"phone_number": types.StringValue(apiData.GetSms().GetPhoneNumber()),
		}, &diags)
	case "whatsapp":
		data.WhatsApp = buildSingleObjList(ctx, whatsappType, map[string]attr.Value{
			"phone_number": types.StringValue(apiData.GetWhatsapp().GetPhoneNumber()),
		}, &diags)
	case "google_chat":
		data.GoogleChat = buildSingleObjList(ctx, googleChatType, map[string]attr.Value{
			"webhook_url": types.StringValue(apiData.GetGoogleChat().GetWebhookUrl()),
		}, &diags)
	case "grafana_oncall":
		data.GrafanaOncall = buildSingleObjList(ctx, grafanaOncallType, map[string]attr.Value{
			"webhook_url": types.StringValue(apiData.GetGrafanaOncall().GetWebhookUrl()),
		}, &diags)
	case "ntfy":
		// The API does not echo back ntfy.token (Sensitive). If the response
		// omits it, fall back to plannedNtfyToken (captured above before the
		// per-provider reset) — otherwise Terraform reports "inconsistent
		// values for sensitive attribute". Preserve the planned shape
		// (null vs value): if the user omitted token in HCL the planned
		// value is null, so post-apply state must be null too.
		var token types.String
		if v := apiData.GetNtfy().GetToken(); v != "" {
			token = types.StringValue(v)
		} else {
			token = plannedNtfyToken
		}
		data.Ntfy = buildSingleObjList(ctx, ntfyType, map[string]attr.Value{
			"topic":      types.StringValue(apiData.GetNtfy().GetTopic()),
			"server_url": types.StringValue(apiData.GetNtfy().GetServerUrl()),
			"token":      token,
		}, &diags)
	case "ms_teams":
		data.MSTeams = buildSingleObjList(ctx, msTeamsType, map[string]attr.Value{
			"webhook_url": types.StringValue(apiData.GetMsTeams().GetWebhookUrl()),
		}, &diags)
	}

	return diags
}

func buildSingleObjList(_ context.Context, attrTypes map[string]attr.Type, vals map[string]attr.Value, diags *diag.Diagnostics) types.List {
	obj, d := types.ObjectValue(attrTypes, vals)
	diags.Append(d...)
	list, d := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{obj})
	diags.Append(d...)
	return list
}

// existingNtfyStringField returns the planned/state value of the given
// tfsdk string attribute on the single-element ntfy list. Returns
// types.StringNull() if the list is null/empty/unknown or the attribute
// isn't a string. Preserves the *shape* (null vs value vs empty string)
// — critical for Sensitive attributes, where Terraform compares plan and
// post-apply state byte-for-byte.
func existingNtfyStringField(list types.List, attr string) types.String {
	if list.IsNull() || list.IsUnknown() {
		return types.StringNull()
	}
	elems := list.Elements()
	if len(elems) == 0 {
		return types.StringNull()
	}
	obj, ok := elems[0].(types.Object)
	if !ok {
		return types.StringNull()
	}
	v, ok := obj.Attributes()[attr]
	if !ok {
		return types.StringNull()
	}
	s, ok := v.(types.String)
	if !ok {
		return types.StringNull()
	}
	return s
}
