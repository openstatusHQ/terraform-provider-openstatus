package monitor

import (
	"fmt"
	"sort"
	"strings"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
)

var periodicityToAPI = map[string]monitorv1.Periodicity{
	"30s": monitorv1.Periodicity_PERIODICITY_30S,
	"1m":  monitorv1.Periodicity_PERIODICITY_1M,
	"5m":  monitorv1.Periodicity_PERIODICITY_5M,
	"10m": monitorv1.Periodicity_PERIODICITY_10M,
	"30m": monitorv1.Periodicity_PERIODICITY_30M,
	"1h":  monitorv1.Periodicity_PERIODICITY_1H,
}

var periodicityFromAPI = reverseMap(periodicityToAPI)

var PeriodicityValues = keys(periodicityToAPI)

var regionToAPI = map[string]monitorv1.Region{
	"fly-ams":                 monitorv1.Region_REGION_FLY_AMS,
	"fly-arn":                 monitorv1.Region_REGION_FLY_ARN,
	"fly-bom":                 monitorv1.Region_REGION_FLY_BOM,
	"fly-cdg":                 monitorv1.Region_REGION_FLY_CDG,
	"fly-dfw":                 monitorv1.Region_REGION_FLY_DFW,
	"fly-ewr":                 monitorv1.Region_REGION_FLY_EWR,
	"fly-fra":                 monitorv1.Region_REGION_FLY_FRA,
	"fly-gru":                 monitorv1.Region_REGION_FLY_GRU,
	"fly-iad":                 monitorv1.Region_REGION_FLY_IAD,
	"fly-jnb":                 monitorv1.Region_REGION_FLY_JNB,
	"fly-lax":                 monitorv1.Region_REGION_FLY_LAX,
	"fly-lhr":                 monitorv1.Region_REGION_FLY_LHR,
	"fly-nrt":                 monitorv1.Region_REGION_FLY_NRT,
	"fly-ord":                 monitorv1.Region_REGION_FLY_ORD,
	"fly-sjc":                 monitorv1.Region_REGION_FLY_SJC,
	"fly-sin":                 monitorv1.Region_REGION_FLY_SIN,
	"fly-syd":                 monitorv1.Region_REGION_FLY_SYD,
	"fly-yyz":                 monitorv1.Region_REGION_FLY_YYZ,
	"koyeb-fra":               monitorv1.Region_REGION_KOYEB_FRA,
	"koyeb-par":               monitorv1.Region_REGION_KOYEB_PAR,
	"koyeb-sfo":               monitorv1.Region_REGION_KOYEB_SFO,
	"koyeb-sin":               monitorv1.Region_REGION_KOYEB_SIN,
	"koyeb-tyo":               monitorv1.Region_REGION_KOYEB_TYO,
	"koyeb-was":               monitorv1.Region_REGION_KOYEB_WAS,
	"railway-us-west2":        monitorv1.Region_REGION_RAILWAY_US_WEST2,
	"railway-us-east4":        monitorv1.Region_REGION_RAILWAY_US_EAST4,
	"railway-europe-west4":    monitorv1.Region_REGION_RAILWAY_EUROPE_WEST4,
	"railway-asia-southeast1": monitorv1.Region_REGION_RAILWAY_ASIA_SOUTHEAST1,
}

var regionFromAPI = reverseMap(regionToAPI)

var RegionValues = keys(regionToAPI)

var methodToAPI = map[string]monitorv1.HTTPMethod{
	"GET":     monitorv1.HTTPMethod_HTTP_METHOD_GET,
	"POST":    monitorv1.HTTPMethod_HTTP_METHOD_POST,
	"HEAD":    monitorv1.HTTPMethod_HTTP_METHOD_HEAD,
	"PUT":     monitorv1.HTTPMethod_HTTP_METHOD_PUT,
	"PATCH":   monitorv1.HTTPMethod_HTTP_METHOD_PATCH,
	"DELETE":  monitorv1.HTTPMethod_HTTP_METHOD_DELETE,
	"TRACE":   monitorv1.HTTPMethod_HTTP_METHOD_TRACE,
	"CONNECT": monitorv1.HTTPMethod_HTTP_METHOD_CONNECT,
	"OPTIONS": monitorv1.HTTPMethod_HTTP_METHOD_OPTIONS,
}

var methodFromAPI = reverseMap(methodToAPI)

var MethodValues = keys(methodToAPI)

var numberComparatorToAPI = map[string]monitorv1.NumberComparator{
	"eq":  monitorv1.NumberComparator_NUMBER_COMPARATOR_EQUAL,
	"neq": monitorv1.NumberComparator_NUMBER_COMPARATOR_NOT_EQUAL,
	"gt":  monitorv1.NumberComparator_NUMBER_COMPARATOR_GREATER_THAN,
	"gte": monitorv1.NumberComparator_NUMBER_COMPARATOR_GREATER_THAN_OR_EQUAL,
	"lt":  monitorv1.NumberComparator_NUMBER_COMPARATOR_LESS_THAN,
	"lte": monitorv1.NumberComparator_NUMBER_COMPARATOR_LESS_THAN_OR_EQUAL,
}

var numberComparatorFromAPI = reverseMap(numberComparatorToAPI)

var NumberComparatorValues = keys(numberComparatorToAPI)

var stringComparatorToAPI = map[string]monitorv1.StringComparator{
	"contains":     monitorv1.StringComparator_STRING_COMPARATOR_CONTAINS,
	"not_contains": monitorv1.StringComparator_STRING_COMPARATOR_NOT_CONTAINS,
	"eq":           monitorv1.StringComparator_STRING_COMPARATOR_EQUAL,
	"neq":          monitorv1.StringComparator_STRING_COMPARATOR_NOT_EQUAL,
	"empty":        monitorv1.StringComparator_STRING_COMPARATOR_EMPTY,
	"not_empty":    monitorv1.StringComparator_STRING_COMPARATOR_NOT_EMPTY,
	"gt":           monitorv1.StringComparator_STRING_COMPARATOR_GREATER_THAN,
	"gte":          monitorv1.StringComparator_STRING_COMPARATOR_GREATER_THAN_OR_EQUAL,
	"lt":           monitorv1.StringComparator_STRING_COMPARATOR_LESS_THAN,
	"lte":          monitorv1.StringComparator_STRING_COMPARATOR_LESS_THAN_OR_EQUAL,
}

var stringComparatorFromAPI = reverseMap(stringComparatorToAPI)

var StringComparatorValues = keys(stringComparatorToAPI)

var recordComparatorToAPI = map[string]monitorv1.RecordComparator{
	"eq":           monitorv1.RecordComparator_RECORD_COMPARATOR_EQUAL,
	"neq":          monitorv1.RecordComparator_RECORD_COMPARATOR_NOT_EQUAL,
	"contains":     monitorv1.RecordComparator_RECORD_COMPARATOR_CONTAINS,
	"not_contains": monitorv1.RecordComparator_RECORD_COMPARATOR_NOT_CONTAINS,
}

var recordComparatorFromAPI = reverseMap(recordComparatorToAPI)

var RecordComparatorValues = keys(recordComparatorToAPI)

var DNSRecordTypes = []string{"A", "AAAA", "CNAME", "MX", "TXT"}

var monitorStatusFromAPI = map[monitorv1.MonitorStatus]string{
	monitorv1.MonitorStatus_MONITOR_STATUS_ACTIVE:   "active",
	monitorv1.MonitorStatus_MONITOR_STATUS_DEGRADED: "degraded",
	monitorv1.MonitorStatus_MONITOR_STATUS_ERROR:    "error",
}

func MapPeriodicityToAPI(v string) (monitorv1.Periodicity, error) {
	return lookupOrError(periodicityToAPI, v, "periodicity")
}

func MapPeriodicityFromAPI(v monitorv1.Periodicity) string {
	if s, ok := periodicityFromAPI[v]; ok {
		return s
	}
	return strings.ToLower(v.String())
}

func MapRegionsToAPI(regions []string) ([]monitorv1.Region, error) {
	out := make([]monitorv1.Region, 0, len(regions))
	for _, r := range regions {
		v, err := lookupOrError(regionToAPI, r, "region")
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func MapRegionsFromAPI(regions []monitorv1.Region) []string {
	out := make([]string, 0, len(regions))
	for _, r := range regions {
		if s, ok := regionFromAPI[r]; ok {
			out = append(out, s)
		}
	}
	return out
}

func MapMethodToAPI(v string) (monitorv1.HTTPMethod, error) {
	return lookupOrError(methodToAPI, v, "method")
}

func MapMethodFromAPI(v monitorv1.HTTPMethod) string {
	if s, ok := methodFromAPI[v]; ok {
		return s
	}
	return v.String()
}

func MapNumberComparatorToAPI(v string) (monitorv1.NumberComparator, error) {
	return lookupOrError(numberComparatorToAPI, v, "number comparator")
}

func MapNumberComparatorFromAPI(v monitorv1.NumberComparator) string {
	if s, ok := numberComparatorFromAPI[v]; ok {
		return s
	}
	return v.String()
}

func MapStringComparatorToAPI(v string) (monitorv1.StringComparator, error) {
	return lookupOrError(stringComparatorToAPI, v, "string comparator")
}

func MapStringComparatorFromAPI(v monitorv1.StringComparator) string {
	if s, ok := stringComparatorFromAPI[v]; ok {
		return s
	}
	return v.String()
}

func MapRecordComparatorToAPI(v string) (monitorv1.RecordComparator, error) {
	return lookupOrError(recordComparatorToAPI, v, "record comparator")
}

func MapRecordComparatorFromAPI(v monitorv1.RecordComparator) string {
	if s, ok := recordComparatorFromAPI[v]; ok {
		return s
	}
	return v.String()
}

func MapMonitorStatusFromAPI(v monitorv1.MonitorStatus) string {
	if s, ok := monitorStatusFromAPI[v]; ok {
		return s
	}
	return "unknown"
}

func lookupOrError[T any](m map[string]T, key, label string) (T, error) {
	v, ok := m[key]
	if !ok {
		var zero T
		return zero, fmt.Errorf("unknown %s: %q", label, key)
	}
	return v, nil
}

func reverseMap[T comparable](m map[string]T) map[T]string {
	out := make(map[T]string, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}

func keys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
