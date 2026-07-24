package statuspage

import (
	"testing"

	statuspagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func monitorComponent(order, groupOrder int32) *statuspagev1.PageComponent {
	c := &statuspagev1.PageComponent{}
	c.SetId("1")
	c.SetPageId("2")
	c.SetName("Acme API")
	c.SetType(statuspagev1.PageComponentType_PAGE_COMPONENT_TYPE_MONITOR)
	c.SetMonitorId("42")
	c.SetGroupId("10")
	c.SetOrder(order)
	c.SetGroupOrder(groupOrder)
	c.SetCreatedAt("2026-03-23T00:00:00Z")
	c.SetUpdatedAt("2026-03-23T00:00:00Z")
	return c
}

func TestAddMonitorComponentRequest_SetsOrderOnlyWhenKnown(t *testing.T) {
	req := &statuspagev1.AddMonitorComponentRequest{}
	req.SetPageId("2")
	req.SetMonitorId("42")
	if req.HasOrder() {
		t.Error("order must be absent until explicitly set")
	}

	req.SetOrder(3)
	if !req.HasOrder() {
		t.Fatal("order must be present once set")
	}
	if req.GetOrder() != 3 {
		t.Errorf("order = %d, want 3", req.GetOrder())
	}
}

func TestUpdateComponentRequest_PresenceReflectsWhatWasSet(t *testing.T) {
	req := &statuspagev1.UpdateComponentRequest{}
	req.SetId("abc")

	for name, has := range map[string]bool{
		"name":        req.HasName(),
		"description": req.HasDescription(),
		"order":       req.HasOrder(),
		"group_id":    req.HasGroupId(),
		"group_order": req.HasGroupOrder(),
	} {
		if has {
			t.Errorf("%s must be absent when not set, otherwise update clobbers it", name)
		}
	}

	req.SetOrder(0)
	if !req.HasOrder() {
		t.Fatal("order must be present after being set to zero")
	}
	if req.GetOrder() != 0 {
		t.Errorf("order = %d, want 0", req.GetOrder())
	}

	req.SetGroupOrder(7)
	if !req.HasGroupOrder() || req.GetGroupOrder() != 7 {
		t.Errorf("group_order = %d (present=%v), want 7 present", req.GetGroupOrder(), req.HasGroupOrder())
	}
}

func TestComponentAPIToModel_SetsGroupOrder(t *testing.T) {
	var data componentModel
	componentAPIToModel(monitorComponent(0, 3), &data)

	if data.GroupOrder.ValueInt64() != 3 {
		t.Errorf("GroupOrder = %d, want 3", data.GroupOrder.ValueInt64())
	}
	if !data.Order.IsNull() {
		t.Errorf("Order = %v, want null (zero is indistinguishable from absent on the wire)", data.Order)
	}
}

func TestComponentAPIToModel_PreservesOrderWhenAPIReportsZero(t *testing.T) {
	data := componentModel{
		Order:      types.Int64Value(1),
		GroupOrder: types.Int64Value(3),
	}
	componentAPIToModel(monitorComponent(0, 0), &data)

	if data.GroupOrder.ValueInt64() != 3 {
		t.Errorf("GroupOrder = %d, want 3 (preserved when API reports zero)", data.GroupOrder.ValueInt64())
	}
	if data.Order.ValueInt64() != 1 {
		t.Errorf("Order = %d, want 1 (preserved when API reports zero)", data.Order.ValueInt64())
	}
}

func TestComponentAPIToModel_OverwritesGroupOrderWhenAPIReturnsNonZero(t *testing.T) {
	data := componentModel{
		Order:      types.Int64Value(1),
		GroupOrder: types.Int64Value(3),
	}
	componentAPIToModel(monitorComponent(5, 7), &data)

	if data.GroupOrder.ValueInt64() != 7 {
		t.Errorf("GroupOrder = %d, want 7", data.GroupOrder.ValueInt64())
	}
	if data.Order.ValueInt64() != 5 {
		t.Errorf("Order = %d, want 5", data.Order.ValueInt64())
	}
}

func TestComponentAPIToModel_CollapsesUnknownToNullWhenAPIReportsZero(t *testing.T) {
	data := componentModel{
		Order:      types.Int64Unknown(),
		GroupOrder: types.Int64Unknown(),
	}
	componentAPIToModel(monitorComponent(0, 0), &data)

	if !data.GroupOrder.IsNull() {
		t.Errorf("GroupOrder = %v, want Null", data.GroupOrder)
	}
	if !data.Order.IsNull() {
		t.Errorf("Order = %v, want Null", data.Order)
	}
}

func TestComponentAPIToModel_MapsType(t *testing.T) {
	var monitorData componentModel
	componentAPIToModel(monitorComponent(1, 1), &monitorData)
	if monitorData.Type.ValueString() != "monitor" {
		t.Errorf("Type = %q, want monitor", monitorData.Type.ValueString())
	}

	static := monitorComponent(1, 1)
	static.SetType(statuspagev1.PageComponentType_PAGE_COMPONENT_TYPE_STATIC)
	var staticData componentModel
	componentAPIToModel(static, &staticData)
	if staticData.Type.ValueString() != "static" {
		t.Errorf("Type = %q, want static", staticData.Type.ValueString())
	}
}
