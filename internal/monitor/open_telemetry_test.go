package monitor

import (
	"encoding/json"
	"testing"
)

func TestAPIOpenTelemetry_MarshalShape(t *testing.T) {
	otel := apiOpenTelemetry{
		Endpoint: "https://otel.example.com",
		Headers:  []apiHeader{{Key: "X-Api-Key", Value: "secret"}},
	}
	b, err := json.Marshal(otel)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["endpoint"] != "https://otel.example.com" {
		t.Errorf("endpoint = %v, want https://otel.example.com", raw["endpoint"])
	}
	headers, ok := raw["headers"].([]interface{})
	if !ok || len(headers) != 1 {
		t.Fatalf("headers = %v, want one entry", raw["headers"])
	}
	h := headers[0].(map[string]interface{})
	if h["key"] != "X-Api-Key" || h["value"] != "secret" {
		t.Errorf("headers[0] = %v, want {key:X-Api-Key, value:secret}", h)
	}
}

func TestHTTPMonitorAPIObject_OpenTelemetryNullWhenAbsent(t *testing.T) {
	obj := httpMonitorAPIObject{
		Name:        "m",
		URL:         "https://example.com",
		Periodicity: "PERIODICITY_1M",
	}
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	otel, ok := raw["openTelemetry"]
	if !ok {
		t.Fatalf("openTelemetry must be present (sent as null to clear server-side)")
	}
	if string(otel) != "null" {
		t.Errorf("openTelemetry = %s, want null", string(otel))
	}
}

func TestHTTPMonitorAPIObject_OpenTelemetryRoundTrip(t *testing.T) {
	in := httpMonitorAPIObject{
		Name:        "m",
		URL:         "https://example.com",
		Periodicity: "PERIODICITY_1M",
		OpenTelemetry: &apiOpenTelemetry{
			Endpoint: "https://otel.example.com",
			Headers:  []apiHeader{{Key: "k", Value: "v"}},
		},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out httpMonitorAPIObject
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.OpenTelemetry == nil {
		t.Fatal("OpenTelemetry round-tripped to nil")
	}
	if out.OpenTelemetry.Endpoint != "https://otel.example.com" {
		t.Errorf("endpoint = %q, want https://otel.example.com", out.OpenTelemetry.Endpoint)
	}
	if len(out.OpenTelemetry.Headers) != 1 || out.OpenTelemetry.Headers[0].Key != "k" {
		t.Errorf("headers = %v, want [{k,v}]", out.OpenTelemetry.Headers)
	}
}
