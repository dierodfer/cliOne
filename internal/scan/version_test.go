package scan

import "testing"

func TestVersionIsNewer(t *testing.T) {
	cases := []struct {
		latest, installed string
		want              bool
	}{
		// genuine updates
		{"1.1.0", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
		{"14.1.1", "14.1.0", true},
		{"1.10.0", "1.9.0", true}, // numeric, not lexical (10 > 9)
		// equal, including formatting differences that used to false-positive
		{"1.7.0", "1.7.0", false},
		{"1.7", "1.7.0", false},
		{"1.7.0", "1.7", false},
		{"v14.1.0", "14.1.0", false},
		// latest older than installed must not report an update
		{"1.0.0", "1.1.0", false},
		{"14.1.0", "14.1.1", false},
		// leading-v normalization on both sides
		{"v2.0.0", "v1.0.0", true},
		// suffixes stop parsing at the first non-numeric field
		{"1.2.0-beta", "1.2.0", false},
		{"1.3.0-beta", "1.2.0", true},
		// non-numeric schemes fall back to string inequality
		{"2024a", "2023b", true},
		{"stable", "stable", false},
	}
	for _, tc := range cases {
		if got := versionIsNewer(tc.latest, tc.installed); got != tc.want {
			t.Errorf("versionIsNewer(%q, %q) = %v, want %v", tc.latest, tc.installed, got, tc.want)
		}
	}
}
