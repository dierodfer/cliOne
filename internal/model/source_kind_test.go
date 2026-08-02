package model

import (
	"encoding/json"
	"testing"
)

func TestSourceKindRoundTripsByName(t *testing.T) {
	for _, k := range []SourceKind{
		SourceUnknown, SourceHomebrew, SourceCargo, SourceUvTool,
		SourceNpmGlobal, SourceAptDnf, SourceAsdf, SourceManual,
	} {
		b, err := json.Marshal(k)
		if err != nil {
			t.Fatalf("marshal %v: %v", k, err)
		}
		var got SourceKind
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if got != k {
			t.Fatalf("round trip of %v gave %v (json %s)", k, got, b)
		}
	}
}

func TestSourceKindEncodesAsNameNotOrdinal(t *testing.T) {
	// The cache persists this value. Encoding the ordinal would remap every
	// stored entry as soon as a kind is inserted into the const block: adding
	// SourceAsdf ahead of SourceManual once turned cached "manual" entries
	// into "asdf" on read, mislabeling every manually installed tool.
	b, err := json.Marshal(SourceManual)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"manual"` {
		t.Fatalf("got %s, want \"manual\"", b)
	}
}

func TestSourceKindRejectsOrdinalForm(t *testing.T) {
	// A pre-name cache file holds integers. Rejecting them makes the file
	// read as corrupt, so it is rebuilt rather than silently misinterpreted.
	var k SourceKind
	if err := json.Unmarshal([]byte("6"), &k); err == nil {
		t.Fatal("expected an error for the legacy ordinal form")
	}
}

func TestParseSourceKindRejectsUnknownName(t *testing.T) {
	if _, ok := ParseSourceKind("nixpkgs"); ok {
		t.Fatal("expected an unrecognized name to report false")
	}
}
