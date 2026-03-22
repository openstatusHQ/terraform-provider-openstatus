package monitor

import (
	"testing"
)

func TestMapPeriodicityToAPI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"30s", "PERIODICITY_30S", false},
		{"1m", "PERIODICITY_1M", false},
		{"5m", "PERIODICITY_5M", false},
		{"10m", "PERIODICITY_10M", false},
		{"30m", "PERIODICITY_30M", false},
		{"1h", "PERIODICITY_1H", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		result, err := MapPeriodicityToAPI(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("MapPeriodicityToAPI(%q) expected error", tt.input)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("MapPeriodicityToAPI(%q) unexpected error: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("MapPeriodicityToAPI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMapPeriodicityFromAPI(t *testing.T) {
	result := MapPeriodicityFromAPI("PERIODICITY_30S")
	if result != "30s" {
		t.Errorf("MapPeriodicityFromAPI(PERIODICITY_30S) = %q, want 30s", result)
	}
}

func TestMapRegionsRoundtrip(t *testing.T) {
	input := []string{"fly-ams", "fly-iad"}
	apiRegions, err := MapRegionsToAPI(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apiRegions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(apiRegions))
	}

	result := MapRegionsFromAPI(apiRegions)
	if len(result) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(result))
	}

	regionSet := map[string]bool{}
	for _, r := range result {
		regionSet[r] = true
	}
	if !regionSet["fly-ams"] || !regionSet["fly-iad"] {
		t.Errorf("expected fly-ams and fly-iad, got %v", result)
	}
}

func TestMapMethodRoundtrip(t *testing.T) {
	for _, method := range MethodValues {
		api, err := MapMethodToAPI(method)
		if err != nil {
			t.Errorf("MapMethodToAPI(%q) error: %v", method, err)
			continue
		}
		back := MapMethodFromAPI(api)
		if back != method {
			t.Errorf("roundtrip %q -> %q -> %q", method, api, back)
		}
	}
}

func TestMapNumberComparatorRoundtrip(t *testing.T) {
	for _, comp := range NumberComparatorValues {
		api, err := MapNumberComparatorToAPI(comp)
		if err != nil {
			t.Errorf("MapNumberComparatorToAPI(%q) error: %v", comp, err)
			continue
		}
		back := MapNumberComparatorFromAPI(api)
		if back != comp {
			t.Errorf("roundtrip %q -> %q -> %q", comp, api, back)
		}
	}
}

func TestMapStringComparatorRoundtrip(t *testing.T) {
	for _, comp := range StringComparatorValues {
		api, err := MapStringComparatorToAPI(comp)
		if err != nil {
			t.Errorf("MapStringComparatorToAPI(%q) error: %v", comp, err)
			continue
		}
		back := MapStringComparatorFromAPI(api)
		if back != comp {
			t.Errorf("roundtrip %q -> %q -> %q", comp, api, back)
		}
	}
}

func TestMapRecordComparatorRoundtrip(t *testing.T) {
	for _, comp := range RecordComparatorValues {
		api, err := MapRecordComparatorToAPI(comp)
		if err != nil {
			t.Errorf("MapRecordComparatorToAPI(%q) error: %v", comp, err)
			continue
		}
		back := MapRecordComparatorFromAPI(api)
		if back != comp {
			t.Errorf("roundtrip %q -> %q -> %q", comp, api, back)
		}
	}
}

func TestMapMonitorStatusFromAPI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"MONITOR_STATUS_ACTIVE", "active"},
		{"MONITOR_STATUS_DEGRADED", "degraded"},
		{"MONITOR_STATUS_ERROR", "error"},
		{"UNKNOWN", "unknown"},
	}

	for _, tt := range tests {
		result := MapMonitorStatusFromAPI(tt.input)
		if result != tt.expected {
			t.Errorf("MapMonitorStatusFromAPI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
