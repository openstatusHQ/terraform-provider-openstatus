package monitor

import (
	"fmt"
	"sort"
	"strings"
)

var periodicityToAPI = map[string]string{
	"30s": "PERIODICITY_30S",
	"1m":  "PERIODICITY_1M",
	"5m":  "PERIODICITY_5M",
	"10m": "PERIODICITY_10M",
	"30m": "PERIODICITY_30M",
	"1h":  "PERIODICITY_1H",
}

var periodicityFromAPI = reverseMap(periodicityToAPI)

var PeriodicityValues = keys(periodicityToAPI)

var regionToAPI = map[string]string{
	"fly-ams":                 "REGION_FLY_AMS",
	"fly-arn":                 "REGION_FLY_ARN",
	"fly-bom":                 "REGION_FLY_BOM",
	"fly-cdg":                 "REGION_FLY_CDG",
	"fly-dfw":                 "REGION_FLY_DFW",
	"fly-ewr":                 "REGION_FLY_EWR",
	"fly-fra":                 "REGION_FLY_FRA",
	"fly-gru":                 "REGION_FLY_GRU",
	"fly-iad":                 "REGION_FLY_IAD",
	"fly-jnb":                 "REGION_FLY_JNB",
	"fly-lax":                 "REGION_FLY_LAX",
	"fly-lhr":                 "REGION_FLY_LHR",
	"fly-nrt":                 "REGION_FLY_NRT",
	"fly-ord":                 "REGION_FLY_ORD",
	"fly-sjc":                 "REGION_FLY_SJC",
	"fly-sin":                 "REGION_FLY_SIN",
	"fly-syd":                 "REGION_FLY_SYD",
	"fly-yyz":                 "REGION_FLY_YYZ",
	"koyeb-fra":               "REGION_KOYEB_FRA",
	"koyeb-par":               "REGION_KOYEB_PAR",
	"koyeb-sfo":               "REGION_KOYEB_SFO",
	"koyeb-sin":               "REGION_KOYEB_SIN",
	"koyeb-tyo":               "REGION_KOYEB_TYO",
	"koyeb-was":               "REGION_KOYEB_WAS",
	"railway-us-west2":        "REGION_RAILWAY_US_WEST2",
	"railway-us-east4":        "REGION_RAILWAY_US_EAST4",
	"railway-europe-west4":    "REGION_RAILWAY_EUROPE_WEST4",
	"railway-asia-southeast1": "REGION_RAILWAY_ASIA_SOUTHEAST1",
}

var regionFromAPI = reverseMap(regionToAPI)

var RegionValues = keys(regionToAPI)

var methodToAPI = map[string]string{
	"GET":     "HTTP_METHOD_GET",
	"POST":    "HTTP_METHOD_POST",
	"HEAD":    "HTTP_METHOD_HEAD",
	"PUT":     "HTTP_METHOD_PUT",
	"PATCH":   "HTTP_METHOD_PATCH",
	"DELETE":  "HTTP_METHOD_DELETE",
	"TRACE":   "HTTP_METHOD_TRACE",
	"CONNECT": "HTTP_METHOD_CONNECT",
	"OPTIONS": "HTTP_METHOD_OPTIONS",
}

var methodFromAPI = reverseMap(methodToAPI)

var MethodValues = keys(methodToAPI)

var numberComparatorToAPI = map[string]string{
	"eq":  "NUMBER_COMPARATOR_EQUAL",
	"neq": "NUMBER_COMPARATOR_NOT_EQUAL",
	"gt":  "NUMBER_COMPARATOR_GREATER_THAN",
	"gte": "NUMBER_COMPARATOR_GREATER_THAN_OR_EQUAL",
	"lt":  "NUMBER_COMPARATOR_LESS_THAN",
	"lte": "NUMBER_COMPARATOR_LESS_THAN_OR_EQUAL",
}

var numberComparatorFromAPI = reverseMap(numberComparatorToAPI)

var NumberComparatorValues = keys(numberComparatorToAPI)

var stringComparatorToAPI = map[string]string{
	"contains":     "STRING_COMPARATOR_CONTAINS",
	"not_contains": "STRING_COMPARATOR_NOT_CONTAINS",
	"eq":           "STRING_COMPARATOR_EQUAL",
	"neq":          "STRING_COMPARATOR_NOT_EQUAL",
	"empty":        "STRING_COMPARATOR_EMPTY",
	"not_empty":    "STRING_COMPARATOR_NOT_EMPTY",
	"gt":           "STRING_COMPARATOR_GREATER_THAN",
	"gte":          "STRING_COMPARATOR_GREATER_THAN_OR_EQUAL",
	"lt":           "STRING_COMPARATOR_LESS_THAN",
	"lte":          "STRING_COMPARATOR_LESS_THAN_OR_EQUAL",
}

var stringComparatorFromAPI = reverseMap(stringComparatorToAPI)

var StringComparatorValues = keys(stringComparatorToAPI)

var recordComparatorToAPI = map[string]string{
	"eq":           "RECORD_COMPARATOR_EQUAL",
	"neq":          "RECORD_COMPARATOR_NOT_EQUAL",
	"contains":     "RECORD_COMPARATOR_CONTAINS",
	"not_contains": "RECORD_COMPARATOR_NOT_CONTAINS",
}

var recordComparatorFromAPI = reverseMap(recordComparatorToAPI)

var RecordComparatorValues = keys(recordComparatorToAPI)

var DNSRecordTypes = []string{"A", "AAAA", "CNAME", "MX", "TXT"}

var monitorStatusFromAPI = map[string]string{
	"MONITOR_STATUS_ACTIVE":   "active",
	"MONITOR_STATUS_DEGRADED": "degraded",
	"MONITOR_STATUS_ERROR":    "error",
}

func MapPeriodicityToAPI(v string) (string, error) {
	return lookupOrError(periodicityToAPI, v, "periodicity")
}

func MapPeriodicityFromAPI(v string) string {
	if s, ok := periodicityFromAPI[v]; ok {
		return s
	}
	return strings.ToLower(v)
}

func MapRegionsToAPI(regions []string) ([]string, error) {
	out := make([]string, 0, len(regions))
	for _, r := range regions {
		v, err := lookupOrError(regionToAPI, r, "region")
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func MapRegionsFromAPI(regions []string) []string {
	out := make([]string, 0, len(regions))
	for _, r := range regions {
		if s, ok := regionFromAPI[r]; ok {
			out = append(out, s)
		}
	}
	return out
}

func MapMethodToAPI(v string) (string, error) {
	return lookupOrError(methodToAPI, v, "method")
}

func MapMethodFromAPI(v string) string {
	if s, ok := methodFromAPI[v]; ok {
		return s
	}
	return v
}

func MapNumberComparatorToAPI(v string) (string, error) {
	return lookupOrError(numberComparatorToAPI, v, "number comparator")
}

func MapNumberComparatorFromAPI(v string) string {
	if s, ok := numberComparatorFromAPI[v]; ok {
		return s
	}
	return v
}

func MapStringComparatorToAPI(v string) (string, error) {
	return lookupOrError(stringComparatorToAPI, v, "string comparator")
}

func MapStringComparatorFromAPI(v string) string {
	if s, ok := stringComparatorFromAPI[v]; ok {
		return s
	}
	return v
}

func MapRecordComparatorToAPI(v string) (string, error) {
	return lookupOrError(recordComparatorToAPI, v, "record comparator")
}

func MapRecordComparatorFromAPI(v string) string {
	if s, ok := recordComparatorFromAPI[v]; ok {
		return s
	}
	return v
}

func MapMonitorStatusFromAPI(v string) string {
	if s, ok := monitorStatusFromAPI[v]; ok {
		return s
	}
	return "unknown"
}

func lookupOrError(m map[string]string, key, label string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("unknown %s: %q", label, key)
	}
	return v, nil
}

func reverseMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
