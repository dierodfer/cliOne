package model

import (
	"encoding/json"
	"fmt"
)

// SourceKind identifies which package manager (if any) owns an installed binary.
type SourceKind int

const (
	SourceUnknown SourceKind = iota
	SourceHomebrew
	SourceCargo
	SourceUvTool
	SourceNpmGlobal
	SourceAptDnf
	SourceAsdf
	SourceManual
)

func (k SourceKind) String() string {
	switch k {
	case SourceHomebrew:
		return "brew"
	case SourceCargo:
		return "cargo"
	case SourceUvTool:
		return "uv"
	case SourceNpmGlobal:
		return "npm"
	case SourceAptDnf:
		return "apt/dnf"
	case SourceAsdf:
		return "asdf"
	case SourceManual:
		return "manual"
	default:
		return "unknown"
	}
}

// ParseSourceKind is the inverse of String. An unrecognized name reports false
// so callers can decide whether to fail or fall back.
func ParseSourceKind(s string) (SourceKind, bool) {
	for _, k := range []SourceKind{
		SourceUnknown, SourceHomebrew, SourceCargo, SourceUvTool,
		SourceNpmGlobal, SourceAptDnf, SourceAsdf, SourceManual,
	} {
		if k.String() == s {
			return k, true
		}
	}
	return SourceUnknown, false
}

// MarshalJSON writes the kind by name rather than by its integer value.
// SourceKind is persisted in the on-disk cache, and encoding the ordinal would
// silently remap every stored entry the moment a new kind is inserted into the
// const block — a cached "manual" would come back as whatever kind took over
// its number.
func (k SourceKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

// UnmarshalJSON reads the name form written by MarshalJSON. A cache file from
// before names were used holds integers here; those fail to decode, which
// callers treat as an unreadable cache and rebuild from scratch.
func (k *SourceKind) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("source kind must be a name, not %s", b)
	}
	parsed, ok := ParseSourceKind(s)
	if !ok {
		return fmt.Errorf("unknown source kind %q", s)
	}
	*k = parsed
	return nil
}
