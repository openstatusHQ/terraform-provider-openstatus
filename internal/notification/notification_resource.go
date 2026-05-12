package notification

import (
	"context"

	"terraform-provider-openstatus/internal/client"

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

	Discord      types.List `tfsdk:"discord"`
	Email        types.List `tfsdk:"email"`
	Slack        types.List `tfsdk:"slack"`
	PagerDuty    types.List `tfsdk:"pagerduty"`
	Opsgenie     types.List `tfsdk:"opsgenie"`
	Webhook      types.List `tfsdk:"webhook"`
	Telegram     types.List `tfsdk:"telegram"`
	SMS          types.List `tfsdk:"sms"`
	WhatsApp     types.List `tfsdk:"whatsapp"`
	GoogleChat   types.List `tfsdk:"google_chat"`
	GrafanaOncall types.List `tfsdk:"grafana_oncall"`
	Ntfy         types.List `tfsdk:"ntfy"`
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
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a notification channel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Optional: true,
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
		},
	}
}

var providerTypeToAPI = map[string]string{
	"discord":        "NOTIFICATION_PROVIDER_DISCORD",
	"email":          "NOTIFICATION_PROVIDER_EMAIL",
	"slack":          "NOTIFICATION_PROVIDER_SLACK",
	"pagerduty":      "NOTIFICATION_PROVIDER_PAGERDUTY",
	"opsgenie":       "NOTIFICATION_PROVIDER_OPSGENIE",
	"webhook":        "NOTIFICATION_PROVIDER_WEBHOOK",
	"telegram":       "NOTIFICATION_PROVIDER_TELEGRAM",
	"sms":            "NOTIFICATION_PROVIDER_SMS",
	"whatsapp":       "NOTIFICATION_PROVIDER_WHATSAPP",
	"google_chat":    "NOTIFICATION_PROVIDER_GOOGLE_CHAT",
	"grafana_oncall": "NOTIFICATION_PROVIDER_GRAFANA_ONCALL",
	"ntfy":           "NOTIFICATION_PROVIDER_NTFY",
}

var providerTypeFromAPI = func() map[string]string {
	m := make(map[string]string, len(providerTypeToAPI))
	for k, v := range providerTypeToAPI {
		m[v] = k
	}
	return m
}()

var opsgenieRegionToAPI = map[string]string{
	"us": "OPSGENIE_REGION_US",
	"eu": "OPSGENIE_REGION_EU",
}

var opsgenieRegionFromAPI = func() map[string]string {
	m := make(map[string]string, len(opsgenieRegionToAPI))
	for k, v := range opsgenieRegionToAPI {
		m[v] = k
	}
	return m
}()

type apiNotificationCreateRequest struct {
	Name       string                 `json:"name,omitempty"`
	Provider   string                 `json:"provider"`
	Data       map[string]interface{} `json:"data"`
	MonitorIDs []string               `json:"monitorIds,omitempty"`
}

type apiNotificationUpdateRequest struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	MonitorIDs []string               `json:"monitorIds,omitempty"`
}

type apiNotificationResponse struct {
	Notification apiNotification `json:"notification"`
}

type apiNotification struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Provider   string                 `json:"provider"`
	Data       map[string]interface{} `json:"data"`
	MonitorIDs []string               `json:"monitorIds"`
	CreatedAt  string                 `json:"createdAt"`
	UpdatedAt  string                 `json:"updatedAt"`
}

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

	var apiResp apiNotificationResponse
	err := r.client.Do(ctx, "/openstatus.notification.v1.NotificationService/CreateNotification", apiReq, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error creating notification", err.Error())
		return
	}

	resp.Diagnostics.Append(notificationAPIToModel(ctx, apiResp.Notification, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *notificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data notificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp apiNotificationResponse
	err := r.client.Do(ctx, "/openstatus.notification.v1.NotificationService/GetNotification",
		map[string]string{"id": data.ID.ValueString()}, &apiResp)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading notification", err.Error())
		return
	}

	resp.Diagnostics.Append(notificationAPIToModel(ctx, apiResp.Notification, &data)...)
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

	var monitorIDs []string
	if !data.MonitorIDs.IsNull() && !data.MonitorIDs.IsUnknown() {
		resp.Diagnostics.Append(data.MonitorIDs.ElementsAs(ctx, &monitorIDs, false)...)
	}

	updateReq := apiNotificationUpdateRequest{
		ID:         state.ID.ValueString(),
		Name:       data.Name.ValueString(),
		Data:       providerData,
		MonitorIDs: monitorIDs,
	}

	var apiResp apiNotificationResponse
	err := r.client.Do(ctx, "/openstatus.notification.v1.NotificationService/UpdateNotification", updateReq, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error updating notification", err.Error())
		return
	}

	resp.Diagnostics.Append(notificationAPIToModel(ctx, apiResp.Notification, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *notificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data notificationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Do(ctx, "/openstatus.notification.v1.NotificationService/DeleteNotification",
		map[string]string{"id": data.ID.ValueString()}, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting notification", err.Error())
	}
}

func (r *notificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
}

func notificationModelToAPI(ctx context.Context, data notificationModel) (apiNotificationCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	providerAPI, ok := providerTypeToAPI[data.ProviderType.ValueString()]
	if !ok {
		diags.AddError("Invalid provider type", "Unknown provider: "+data.ProviderType.ValueString())
		return apiNotificationCreateRequest{}, diags
	}

	providerData, d := extractProviderData(ctx, data)
	diags.Append(d...)
	if diags.HasError() {
		return apiNotificationCreateRequest{}, diags
	}

	var monitorIDs []string
	if !data.MonitorIDs.IsNull() && !data.MonitorIDs.IsUnknown() {
		diags.Append(data.MonitorIDs.ElementsAs(ctx, &monitorIDs, false)...)
	}

	return apiNotificationCreateRequest{
		Name:       data.Name.ValueString(),
		Provider:   providerAPI,
		Data:       providerData,
		MonitorIDs: monitorIDs,
	}, diags
}

func extractProviderData(ctx context.Context, data notificationModel) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	pt := data.ProviderType.ValueString()

	var inner map[string]interface{}
	var wrapKey string

	switch pt {
	case "discord":
		wrapKey = "discord"
		inner, diags = extractWebhookBlock(ctx, data.Discord, &diags)
	case "email":
		wrapKey = "email"
		inner, diags = extractSingleFieldBlock(ctx, data.Email, "email", "email", &diags)
	case "slack":
		wrapKey = "slack"
		inner, diags = extractWebhookBlock(ctx, data.Slack, &diags)
	case "pagerduty":
		wrapKey = "pagerduty"
		inner, diags = extractSingleFieldBlock(ctx, data.PagerDuty, "integration_key", "integrationKey", &diags)
	case "opsgenie":
		wrapKey = "opsgenie"
		inner, diags = extractOpsgenieBlock(ctx, data.Opsgenie, &diags)
	case "webhook":
		wrapKey = "webhook"
		inner, diags = extractWebhookEndpointBlock(ctx, data.Webhook, &diags)
	case "telegram":
		wrapKey = "telegram"
		inner, diags = extractSingleFieldBlock(ctx, data.Telegram, "chat_id", "chatId", &diags)
	case "sms":
		wrapKey = "sms"
		inner, diags = extractSingleFieldBlock(ctx, data.SMS, "phone_number", "phoneNumber", &diags)
	case "whatsapp":
		wrapKey = "whatsapp"
		inner, diags = extractSingleFieldBlock(ctx, data.WhatsApp, "phone_number", "phoneNumber", &diags)
	case "google_chat":
		wrapKey = "googleChat"
		inner, diags = extractWebhookBlock(ctx, data.GoogleChat, &diags)
	case "grafana_oncall":
		wrapKey = "grafanaOncall"
		inner, diags = extractWebhookBlock(ctx, data.GrafanaOncall, &diags)
	case "ntfy":
		wrapKey = "ntfy"
		inner, diags = extractNtfyBlock(ctx, data.Ntfy, &diags)
	}

	if diags.HasError() {
		return nil, diags
	}
	if wrapKey != "" {
		return map[string]interface{}{wrapKey: inner}, diags
	}

	diags.AddError("Unknown provider", "No data block for provider: "+pt)
	return nil, diags
}

func extractWebhookBlock(ctx context.Context, list types.List, diags *diag.Diagnostics) (map[string]interface{}, diag.Diagnostics) {
	var items []struct {
		WebhookURL string `tfsdk:"webhook_url"`
	}
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	if len(items) == 0 {
		diags.AddError("Missing block", "Expected exactly one block")
		return nil, *diags
	}
	return map[string]interface{}{"webhookUrl": items[0].WebhookURL}, *diags
}

func extractSingleFieldBlock(ctx context.Context, list types.List, tfField, apiField string, diags *diag.Diagnostics) (map[string]interface{}, diag.Diagnostics) {
	elems := list.Elements()
	if len(elems) == 0 {
		diags.AddError("Missing block", "Expected exactly one block")
		return nil, *diags
	}
	obj := elems[0].(types.Object)
	val := obj.Attributes()[tfField].(types.String)
	return map[string]interface{}{apiField: val.ValueString()}, *diags
}

func extractOpsgenieBlock(ctx context.Context, list types.List, diags *diag.Diagnostics) (map[string]interface{}, diag.Diagnostics) {
	var items []struct {
		APIKey string `tfsdk:"api_key"`
		Region string `tfsdk:"region"`
	}
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	if len(items) == 0 {
		diags.AddError("Missing block", "Expected exactly one opsgenie block")
		return nil, *diags
	}
	regionAPI := opsgenieRegionToAPI[items[0].Region]
	return map[string]interface{}{"apiKey": items[0].APIKey, "region": regionAPI}, *diags
}

func extractWebhookEndpointBlock(ctx context.Context, list types.List, diags *diag.Diagnostics) (map[string]interface{}, diag.Diagnostics) {
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
		return nil, *diags
	}
	result := map[string]interface{}{"endpoint": items[0].Endpoint}
	if len(items[0].Headers) > 0 {
		headers := make([]map[string]string, 0, len(items[0].Headers))
		for _, h := range items[0].Headers {
			headers = append(headers, map[string]string{"key": h.Key, "value": h.Value})
		}
		result["headers"] = headers
	}
	return result, *diags
}

func extractNtfyBlock(ctx context.Context, list types.List, diags *diag.Diagnostics) (map[string]interface{}, diag.Diagnostics) {
	var items []struct {
		Topic     string `tfsdk:"topic"`
		ServerURL string `tfsdk:"server_url"`
		Token     string `tfsdk:"token"`
	}
	diags.Append(list.ElementsAs(ctx, &items, false)...)
	if len(items) == 0 {
		diags.AddError("Missing block", "Expected exactly one ntfy block")
		return nil, *diags
	}
	result := map[string]interface{}{"topic": items[0].Topic}
	if items[0].ServerURL != "" {
		result["serverUrl"] = items[0].ServerURL
	}
	if items[0].Token != "" {
		result["token"] = items[0].Token
	}
	return result, *diags
}

func notificationAPIToModel(ctx context.Context, api apiNotification, data *notificationModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.CreatedAt = types.StringValue(api.CreatedAt)
	data.UpdatedAt = types.StringValue(api.UpdatedAt)

	if pt, ok := providerTypeFromAPI[api.Provider]; ok {
		data.ProviderType = types.StringValue(pt)
	}

	if len(api.MonitorIDs) > 0 {
		monitorSet, d := types.SetValueFrom(ctx, types.StringType, api.MonitorIDs)
		diags.Append(d...)
		data.MonitorIDs = monitorSet
	} else {
		data.MonitorIDs = types.SetNull(types.StringType)
	}

	pt := data.ProviderType.ValueString()

	// The API wraps provider data under a discriminator key (e.g. {"email": {"email": "..."}}).
	// Unwrap it to get the inner data object.
	apiData := unwrapProviderData(api.Data, pt)

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

	switch pt {
	case "discord":
		data.Discord = buildSingleObjList(ctx, discordType, map[string]attr.Value{
			"webhook_url": types.StringValue(strFromData(apiData, "webhookUrl")),
		}, &diags)
	case "email":
		data.Email = buildSingleObjList(ctx, emailType, map[string]attr.Value{
			"email": types.StringValue(strFromData(apiData, "email")),
		}, &diags)
	case "slack":
		data.Slack = buildSingleObjList(ctx, slackType, map[string]attr.Value{
			"webhook_url": types.StringValue(strFromData(apiData, "webhookUrl")),
		}, &diags)
	case "pagerduty":
		data.PagerDuty = buildSingleObjList(ctx, pagerdutyType, map[string]attr.Value{
			"integration_key": types.StringValue(strFromData(apiData, "integrationKey")),
		}, &diags)
	case "opsgenie":
		region := strFromData(apiData, "region")
		if r, ok := opsgenieRegionFromAPI[region]; ok {
			region = r
		}
		data.Opsgenie = buildSingleObjList(ctx, opsgenieType, map[string]attr.Value{
			"api_key": types.StringValue(strFromData(apiData, "apiKey")),
			"region":  types.StringValue(region),
		}, &diags)
	case "webhook":
		headersVal := types.ListNull(types.ObjectType{AttrTypes: webhookHeaderObjTypes})
		if rawHeaders, ok := apiData["headers"]; ok {
			if headersList, ok := rawHeaders.([]interface{}); ok && len(headersList) > 0 {
				headerObjs := make([]attr.Value, 0, len(headersList))
				for _, h := range headersList {
					if hMap, ok := h.(map[string]interface{}); ok {
						obj, d := types.ObjectValue(webhookHeaderObjTypes, map[string]attr.Value{
							"key":   types.StringValue(strFromMap(hMap, "key")),
							"value": types.StringValue(strFromMap(hMap, "value")),
						})
						diags.Append(d...)
						headerObjs = append(headerObjs, obj)
					}
				}
				headersVal, _ = types.ListValue(types.ObjectType{AttrTypes: webhookHeaderObjTypes}, headerObjs)
			}
		}
		data.Webhook = buildSingleObjList(ctx, webhookType, map[string]attr.Value{
			"endpoint": types.StringValue(strFromData(apiData, "endpoint")),
			"headers":  headersVal,
		}, &diags)
	case "telegram":
		data.Telegram = buildSingleObjList(ctx, telegramType, map[string]attr.Value{
			"chat_id": types.StringValue(strFromData(apiData, "chatId")),
		}, &diags)
	case "sms":
		data.SMS = buildSingleObjList(ctx, smsType, map[string]attr.Value{
			"phone_number": types.StringValue(strFromData(apiData, "phoneNumber")),
		}, &diags)
	case "whatsapp":
		data.WhatsApp = buildSingleObjList(ctx, whatsappType, map[string]attr.Value{
			"phone_number": types.StringValue(strFromData(apiData, "phoneNumber")),
		}, &diags)
	case "google_chat":
		data.GoogleChat = buildSingleObjList(ctx, googleChatType, map[string]attr.Value{
			"webhook_url": types.StringValue(strFromData(apiData, "webhookUrl")),
		}, &diags)
	case "grafana_oncall":
		data.GrafanaOncall = buildSingleObjList(ctx, grafanaOncallType, map[string]attr.Value{
			"webhook_url": types.StringValue(strFromData(apiData, "webhookUrl")),
		}, &diags)
	case "ntfy":
		data.Ntfy = buildSingleObjList(ctx, ntfyType, map[string]attr.Value{
			"topic":      types.StringValue(strFromData(apiData, "topic")),
			"server_url": types.StringValue(strFromData(apiData, "serverUrl")),
			"token":      types.StringValue(strFromData(apiData, "token")),
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

func strFromData(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func strFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

var providerTypeToWrapKey = map[string]string{
	"discord":        "discord",
	"email":          "email",
	"slack":          "slack",
	"pagerduty":      "pagerduty",
	"opsgenie":       "opsgenie",
	"webhook":        "webhook",
	"telegram":       "telegram",
	"sms":            "sms",
	"whatsapp":       "whatsapp",
	"google_chat":    "googleChat",
	"grafana_oncall": "grafanaOncall",
	"ntfy":           "ntfy",
}

func unwrapProviderData(data map[string]interface{}, providerType string) map[string]interface{} {
	wrapKey, ok := providerTypeToWrapKey[providerType]
	if !ok {
		return data
	}
	if inner, ok := data[wrapKey]; ok {
		if innerMap, ok := inner.(map[string]interface{}); ok {
			return innerMap
		}
	}
	return data
}

func isNotFound(err error) bool {
	if apiErr, ok := err.(*client.APIError); ok {
		return apiErr.Code == "not_found"
	}
	return false
}
