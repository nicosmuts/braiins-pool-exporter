package collector

import "sort"

func copyCounters(source map[string]float64) map[string]float64 {
	copied := make(map[string]float64, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func sortedCounterKeys(counters map[string]float64) []string {
	keys := make([]string, 0, len(counters))
	for key := range counters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
