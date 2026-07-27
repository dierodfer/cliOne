package model

// SourceKind identifies which package manager (if any) owns an installed binary.
type SourceKind int

const (
	SourceUnknown SourceKind = iota
	SourceHomebrew
	SourceCargo
	SourceUvTool
	SourceNpmGlobal
	SourceAptDnf
	SourceManual
)

func (k SourceKind) String() string {
	switch k {
	case SourceHomebrew:
		return "homebrew"
	case SourceCargo:
		return "cargo"
	case SourceUvTool:
		return "uv"
	case SourceNpmGlobal:
		return "npm"
	case SourceAptDnf:
		return "apt/dnf"
	case SourceManual:
		return "manual"
	default:
		return "unknown"
	}
}
