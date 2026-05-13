package monitor

import (
	"encoding/json"
	"testing"
)

func TestTCPMonitorAPIObject_BoolFalseNotOmitted(t *testing.T) {
	obj := tcpMonitorAPIObject{
		Name:        "test",
		URI:         "example.com:443",
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

func TestTCPMonitorAPIObject_ClearableScalarsNotOmitted(t *testing.T) {
	obj := tcpMonitorAPIObject{
		Name:        "m",
		URI:         "example.com:443",
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

func TestGetMonitorResponse_NestedMonitorWrapper_TCP(t *testing.T) {
	apiJSON := `{"monitor":{"tcp":{"name":"Orbula","uri":"v3.license.containous.cloud:443"}}}`

	var resp getMonitorResponse
	if err := json.Unmarshal([]byte(apiJSON), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if resp.Monitor.TCP == nil {
		t.Fatal("expected monitor.tcp to be parsed, got nil")
	}
	if resp.Monitor.TCP.Name != "Orbula" {
		t.Errorf("name = %q, want %q", resp.Monitor.TCP.Name, "Orbula")
	}
}
