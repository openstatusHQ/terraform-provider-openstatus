package statuspage

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ptr[T any](v T) *T { return &v }

func TestAPIAddMonitorComponentRequest_IncludesGroupOrder(t *testing.T) {
	req := apiAddMonitorComponentRequest{
		PageID:     "1",
		MonitorID:  "42",
		Name:       "Hub UI",
		Order:      ptr(int64(0)),
		GroupID:    "10",
		GroupOrder: ptr(int64(3)),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	val, ok := raw["groupOrder"]
	if !ok {
		t.Fatal("groupOrder field missing from JSON")
	}
	if val != float64(3) {
		t.Errorf("groupOrder = %v, want 3", val)
	}
}

func TestAPIAddMonitorComponentRequest_OmitsNilGroupOrder(t *testing.T) {
	req := apiAddMonitorComponentRequest{
		PageID:    "1",
		MonitorID: "42",
		Name:      "Hub UI",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if _, ok := raw["groupOrder"]; ok {
		t.Error("groupOrder should be omitted when nil")
	}
	if _, ok := raw["order"]; ok {
		t.Error("order should be omitted when nil")
	}
}

func TestAPIAddStaticComponentRequest_IncludesGroupOrder(t *testing.T) {
	req := apiAddStaticComponentRequest{
		PageID:     "1",
		Name:       "Static",
		Order:      ptr(int64(1)),
		GroupID:    "10",
		GroupOrder: ptr(int64(2)),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	val, ok := raw["groupOrder"]
	if !ok {
		t.Fatal("groupOrder field missing from JSON")
	}
	if val != float64(2) {
		t.Errorf("groupOrder = %v, want 2", val)
	}
}

func TestAPIComponent_GroupOrderParsed(t *testing.T) {
	apiJSON := `{
		"id": "1",
		"pageId": "2",
		"name": "Hub UI",
		"description": "",
		"type": "PAGE_COMPONENT_TYPE_MONITOR",
		"monitorId": "42",
		"order": 0,
		"groupId": "10",
		"groupOrder": 3,
		"createdAt": "2026-03-23T00:00:00Z",
		"updatedAt": "2026-03-23T00:00:00Z"
	}`

	var comp apiComponent
	if err := json.Unmarshal([]byte(apiJSON), &comp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if comp.GroupOrder == nil || *comp.GroupOrder != 3 {
		t.Errorf("GroupOrder = %v, want 3", comp.GroupOrder)
	}
	if comp.Order == nil || *comp.Order != 0 {
		t.Errorf("Order = %v, want 0", comp.Order)
	}
}

func TestAPIUpdateComponentRequest_IncludesGroupOrder(t *testing.T) {
	req := apiUpdateComponentRequest{
		ID:         "1",
		GroupOrder: ptr(int64(5)),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	val, ok := raw["groupOrder"]
	if !ok {
		t.Fatal("groupOrder field missing from JSON")
	}
	if val != float64(5) {
		t.Errorf("groupOrder = %v, want 5", val)
	}
}

func TestAPIUpdateComponentRequest_OmitsNilGroupOrder(t *testing.T) {
	req := apiUpdateComponentRequest{
		ID: "1",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if _, ok := raw["groupOrder"]; ok {
		t.Error("groupOrder should be omitted when nil")
	}
}

func TestComponentAPIToModel_SetsGroupOrder(t *testing.T) {
	api := apiComponent{
		ID:         "1",
		PageID:     "2",
		Name:       "Acme API",
		Type:       "PAGE_COMPONENT_TYPE_MONITOR",
		MonitorID:  "42",
		Order:      ptr(int64(0)),
		GroupID:    "10",
		GroupOrder: ptr(int64(3)),
		CreatedAt:  "2026-03-23T00:00:00Z",
		UpdatedAt:  "2026-03-23T00:00:00Z",
	}

	var data componentModel
	componentAPIToModel(api, &data)

	if data.GroupOrder.ValueInt64() != 3 {
		t.Errorf("GroupOrder = %d, want 3", data.GroupOrder.ValueInt64())
	}
	if data.Order.ValueInt64() != 0 {
		t.Errorf("Order = %d, want 0", data.Order.ValueInt64())
	}
}

func TestComponentAPIToModel_PreservesOrderWhenAPIOmitsFields(t *testing.T) {
	api := apiComponent{
		ID:        "1",
		PageID:    "2",
		Name:      "Acme API",
		Type:      "PAGE_COMPONENT_TYPE_MONITOR",
		MonitorID: "42",
		GroupID:   "10",
		CreatedAt: "2026-03-23T00:00:00Z",
		UpdatedAt: "2026-03-23T00:00:00Z",
		// Order and GroupOrder are nil (omitted by API)
	}

	data := componentModel{
		Order:      types.Int64Value(1),
		GroupOrder: types.Int64Value(3),
	}

	componentAPIToModel(api, &data)

	if data.GroupOrder.ValueInt64() != 3 {
		t.Errorf("GroupOrder = %d, want 3 (should be preserved when API omits field)", data.GroupOrder.ValueInt64())
	}
	if data.Order.ValueInt64() != 1 {
		t.Errorf("Order = %d, want 1 (should be preserved when API omits field)", data.Order.ValueInt64())
	}
}

func TestComponentAPIToModel_SetsExplicitZero(t *testing.T) {
	api := apiComponent{
		ID:         "1",
		PageID:     "2",
		Name:       "Acme API",
		Type:       "PAGE_COMPONENT_TYPE_MONITOR",
		MonitorID:  "42",
		Order:      ptr(int64(0)),
		GroupID:    "10",
		GroupOrder: ptr(int64(0)),
		CreatedAt:  "2026-03-23T00:00:00Z",
		UpdatedAt:  "2026-03-23T00:00:00Z",
	}

	data := componentModel{
		Order:      types.Int64Value(1),
		GroupOrder: types.Int64Value(3),
	}

	componentAPIToModel(api, &data)

	if data.GroupOrder.ValueInt64() != 0 {
		t.Errorf("GroupOrder = %d, want 0 (explicit zero from API)", data.GroupOrder.ValueInt64())
	}
	if data.Order.ValueInt64() != 0 {
		t.Errorf("Order = %d, want 0 (explicit zero from API)", data.Order.ValueInt64())
	}
}

func TestComponentAPIToModel_OverwritesGroupOrderWhenAPIReturnsNonZero(t *testing.T) {
	api := apiComponent{
		ID:         "1",
		PageID:     "2",
		Name:       "Acme API",
		Type:       "PAGE_COMPONENT_TYPE_MONITOR",
		MonitorID:  "42",
		Order:      ptr(int64(5)),
		GroupID:    "10",
		GroupOrder: ptr(int64(7)),
		CreatedAt:  "2026-03-23T00:00:00Z",
		UpdatedAt:  "2026-03-23T00:00:00Z",
	}

	data := componentModel{
		Order:      types.Int64Value(1),
		GroupOrder: types.Int64Value(3),
	}

	componentAPIToModel(api, &data)

	if data.GroupOrder.ValueInt64() != 7 {
		t.Errorf("GroupOrder = %d, want 7", data.GroupOrder.ValueInt64())
	}
	if data.Order.ValueInt64() != 5 {
		t.Errorf("Order = %d, want 5", data.Order.ValueInt64())
	}
}
