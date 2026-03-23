package monitor

import (
	"encoding/json"
	"testing"
)

func TestHTTPMonitorAPIObject_BoolFalseNotOmitted(t *testing.T) {
	obj := httpMonitorAPIObject{
		Name:            "test",
		URL:             "https://example.com",
		Periodicity:     "PERIODICITY_1M",
		FollowRedirects: false,
		Active:          false,
		Public:          false,
	}

	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	for _, field := range []string{"followRedirects", "active", "public"} {
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

func TestGetMonitorResponse_NestedMonitorWrapper(t *testing.T) {
	// Simulates the actual API v2 response format
	apiJSON := `{"monitor":{"http":{"name":"Hub UI","url":"https://hub.traefik.io","followRedirects":false}}}`

	var resp getMonitorResponse
	if err := json.Unmarshal([]byte(apiJSON), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if resp.Monitor.HTTP == nil {
		t.Fatal("expected monitor.http to be parsed, got nil")
	}
	if resp.Monitor.HTTP.Name != "Hub UI" {
		t.Errorf("name = %q, want %q", resp.Monitor.HTTP.Name, "Hub UI")
	}
	if resp.Monitor.HTTP.FollowRedirects != false {
		t.Error("followRedirects = true, want false")
	}
}
