package scan

import (
	"strconv"
	"strings"
)

// versionIsNewer reports whether latest is a strictly newer version than
// installed, using a dotted-numeric comparison rather than raw string
// inequality. This avoids two classes of false "update available":
//
//   - formatting differences between equal versions (e.g. "1.7" vs "1.7.0"),
//   - a latest that is actually older than what is installed.
//
// Both inputs are normalized (leading "v" and surrounding space stripped) and
// split on ".". Each field is compared by its leading integer, with missing
// trailing fields treated as 0. If neither side yields any numeric field, it
// falls back to a plain string inequality so unusual version schemes still
// surface a difference.
func versionIsNewer(latest, installed string) bool {
	l := parseVersion(latest)
	i := parseVersion(installed)
	if len(l) == 0 && len(i) == 0 {
		return normalizeVer(latest) != normalizeVer(installed)
	}
	n := len(l)
	if len(i) > n {
		n = len(i)
	}
	for k := 0; k < n; k++ {
		var lv, iv int
		if k < len(l) {
			lv = l[k]
		}
		if k < len(i) {
			iv = i[k]
		}
		if lv != iv {
			return lv > iv
		}
	}
	return false
}

// parseVersion splits a normalized version into its leading-integer fields.
// A field with no leading digits (e.g. a "-beta" suffix) stops parsing.
func parseVersion(v string) []int {
	v = normalizeVer(v)
	if v == "" {
		return nil
	}
	var out []int
	for _, field := range strings.Split(v, ".") {
		n := leadingInt(field)
		if n < 0 {
			break
		}
		out = append(out, n)
	}
	return out
}

// normalizeVer trims surrounding space and a single leading "v".
func normalizeVer(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// leadingInt returns the integer formed by the leading digits of s, or -1 when
// s has no leading digit.
func leadingInt(s string) int {
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return -1
	}
	n, err := strconv.Atoi(s[:end])
	if err != nil {
		return -1
	}
	return n
}
