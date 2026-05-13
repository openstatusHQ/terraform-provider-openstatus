package monitor

import (
	"encoding/json"
	"testing"
)

func TestDNSMonitorAPIObject_BoolFalseNotOmitted(t *testing.T) {
	obj := dnsMonitorAPIObject{
		Name:        "test",
		URI:         "example.com",
		Periodicity: "PERIODICITY_1M",
		Active:      false,
		Public:      false,
	}

	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	for _, field := range []string{"active", "public"} {
		val, ok := raw[field]
		if !ok {
			t.Errorf("field %q omitted from JSON when false — omitempty must be removed", field)
			continue
		}
		if val != false {
			t.Errorf("field %q = %v, want false", field, val)
		}
	}
}

func TestDNSMonitorAPIObject_ClearableScalarsNotOmitted(t *testing.T) {
	obj := dnsMonitorAPIObject{
		Name:        "m",
		URI:         "example.com",
		Periodicity: "PERIODICITY_1M",
		Description: "",
		Timeout:     0,
		Retry:       0,
	}
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, field := range []string{"description", "timeout", "retry"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("field %q omitted from JSON when zero — omitempty must be removed", field)
		}
	}
}

func TestGetMonitorResponse_NestedMonitorWrapper_DNS(t *testing.T) {
	apiJSON := `{"monitor":{"dns":{"name":"DNS Check","uri":"example.com"}}}`

	var resp getMonitorResponse
	if err := json.Unmarshal([]byte(apiJSON), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if resp.Monitor.DNS == nil {
		t.Fatal("expected monitor.dns to be parsed, got nil")
	}
	if resp.Monitor.DNS.Name != "DNS Check" {
		t.Errorf("name = %q, want %q", resp.Monitor.DNS.Name, "DNS Check")
	}
}
