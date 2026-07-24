package monitor

import (
	"sort"
	"strings"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
)

func TestMapPeriodicityToAPI(t *testing.T) {
	tests := []struct {
		input    string
		expected monitorv1.Periodicity
		wantErr  bool
	}{
		{"30s", monitorv1.Periodicity_PERIODICITY_30S, false},
		{"1m", monitorv1.Periodicity_PERIODICITY_1M, false},
		{"5m", monitorv1.Periodicity_PERIODICITY_5M, false},
		{"10m", monitorv1.Periodicity_PERIODICITY_10M, false},
		{"30m", monitorv1.Periodicity_PERIODICITY_30M, false},
		{"1h", monitorv1.Periodicity_PERIODICITY_1H, false},
		{"invalid", monitorv1.Periodicity_PERIODICITY_UNSPECIFIED, true},
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
			t.Errorf("MapPeriodicityToAPI(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestMapPeriodicityFromAPI(t *testing.T) {
	result := MapPeriodicityFromAPI(monitorv1.Periodicity_PERIODICITY_30S)
	if result != "30s" {
		t.Errorf("MapPeriodicityFromAPI(PERIODICITY_30S) = %q, want 30s", result)
	}
}

func TestRegionValuesSorted(t *testing.T) {
	if !sort.StringsAreSorted(RegionValues) {
		t.Errorf("RegionValues must be sorted for deterministic docs; got %v", RegionValues)
	}
}

func TestMapRegionsToAPI_InvalidReturnsError(t *testing.T) {
	_, err := MapRegionsToAPI([]string{"railway-europe-ewlrljw"})
	if err == nil {
		t.Fatal("expected error for unknown region, got nil")
	}
	if !strings.Contains(err.Error(), `unknown region: "railway-europe-ewlrljw"`) {
		t.Errorf("unexpected error message: %v", err)
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
		input    monitorv1.MonitorStatus
		expected string
	}{
		{monitorv1.MonitorStatus_MONITOR_STATUS_ACTIVE, "active"},
		{monitorv1.MonitorStatus_MONITOR_STATUS_DEGRADED, "degraded"},
		{monitorv1.MonitorStatus_MONITOR_STATUS_ERROR, "error"},
		{monitorv1.MonitorStatus(99), "unknown"},
	}

	for _, tt := range tests {
		result := MapMonitorStatusFromAPI(tt.input)
		if result != tt.expected {
			t.Errorf("MapMonitorStatusFromAPI(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestEnumMapsCoverEveryGeneratedValue(t *testing.T) {
	for value, name := range monitorv1.Region_name {
		region := monitorv1.Region(value)
		if region == monitorv1.Region_REGION_UNSPECIFIED {
			continue
		}
		if _, ok := regionFromAPI[region]; !ok {
			t.Errorf("generated region %s is not mapped to a Terraform value", name)
		}
	}
	for value, name := range monitorv1.Periodicity_name {
		p := monitorv1.Periodicity(value)
		if p == monitorv1.Periodicity_PERIODICITY_UNSPECIFIED {
			continue
		}
		if _, ok := periodicityFromAPI[p]; !ok {
			t.Errorf("generated periodicity %s is not mapped to a Terraform value", name)
		}
	}
	for value, name := range monitorv1.HTTPMethod_name {
		m := monitorv1.HTTPMethod(value)
		if m == monitorv1.HTTPMethod_HTTP_METHOD_UNSPECIFIED {
			continue
		}
		if _, ok := methodFromAPI[m]; !ok {
			t.Errorf("generated method %s is not mapped to a Terraform value", name)
		}
	}
	for value, name := range monitorv1.NumberComparator_name {
		c := monitorv1.NumberComparator(value)
		if c == monitorv1.NumberComparator_NUMBER_COMPARATOR_UNSPECIFIED {
			continue
		}
		if _, ok := numberComparatorFromAPI[c]; !ok {
			t.Errorf("generated number comparator %s is not mapped to a Terraform value", name)
		}
	}
	for value, name := range monitorv1.StringComparator_name {
		c := monitorv1.StringComparator(value)
		if c == monitorv1.StringComparator_STRING_COMPARATOR_UNSPECIFIED {
			continue
		}
		if _, ok := stringComparatorFromAPI[c]; !ok {
			t.Errorf("generated string comparator %s is not mapped to a Terraform value", name)
		}
	}
	for value, name := range monitorv1.RecordComparator_name {
		c := monitorv1.RecordComparator(value)
		if c == monitorv1.RecordComparator_RECORD_COMPARATOR_UNSPECIFIED {
			continue
		}
		if _, ok := recordComparatorFromAPI[c]; !ok {
			t.Errorf("generated record comparator %s is not mapped to a Terraform value", name)
		}
	}
	for value, name := range monitorv1.MonitorStatus_name {
		st := monitorv1.MonitorStatus(value)
		if st == monitorv1.MonitorStatus_MONITOR_STATUS_UNSPECIFIED {
			continue
		}
		if _, ok := monitorStatusFromAPI[st]; !ok {
			t.Errorf("generated monitor status %s is not mapped to a Terraform value", name)
		}
	}
}
