package models

import "strings"

var knownStatusPrefixes = []string{"cs_", "qs_", "qm_", "qms_", "qmm_"}

func NormalizeSessionStatus(status string) string {
	st := strings.TrimSpace(status)
	for _, p := range knownStatusPrefixes {
		if strings.HasPrefix(st, p) {
			return strings.TrimPrefix(st, p)
		}
	}
	return st
}

func ApplyStatusPrefix(fromStatus string, toLegacyStatus string) string {
	fs := strings.TrimSpace(fromStatus)
	ts := strings.TrimSpace(toLegacyStatus)
	for _, p := range knownStatusPrefixes {
		if strings.HasPrefix(fs, p) {
			if strings.HasPrefix(ts, p) {
				return ts
			}
			return p + ts
		}
	}
	return ts
}

func WithCSPrefix(status string) string {
	st := strings.TrimSpace(status)
	if st == "" {
		return ""
	}
	if strings.HasPrefix(st, "cs_") {
		return st
	}
	return "cs_" + st
}

func WithModePrefix(mode string, status string) string {
	m := strings.TrimSpace(mode)
	st := strings.TrimSpace(status)
	if st == "" {
		return ""
	}
	for _, p := range knownStatusPrefixes {
		if strings.HasPrefix(st, p) {
			return st
		}
	}
	if m == "" {
		return st
	}
	return m + "_" + st
}

func ExpandStatusWithKnownPrefixes(status string) []string {
	st := strings.TrimSpace(status)
	if st == "" {
		return []string{}
	}
	base := NormalizeSessionStatus(st)
	out := make([]string, 0, 1+len(knownStatusPrefixes))
	out = append(out, base)
	for _, p := range knownStatusPrefixes {
		out = append(out, p+base)
	}
	return out
}

func ExpandStatusesWithKnownPrefixes(statuses []string) []string {
	if len(statuses) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(statuses)*(1+len(knownStatusPrefixes)))
	for _, st := range statuses {
		for _, s := range ExpandStatusWithKnownPrefixes(st) {
			if s == "" {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
