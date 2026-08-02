// Package model holds CLIOne's shared domain types.
//
// It is the vocabulary shared by every other internal package and must not
// import any other internal package.
package model

// ToolDef describes a tool tracked by CLIOne, as declared in the catalog.
type ToolDef struct {
	ID       string     `yaml:"id"`
	Name     string     `yaml:"name"`
	Category string     `yaml:"category"`
	Detect   DetectSpec `yaml:"detect"`
	// PackageName is the name the owning package manager knows the tool by,
	// when it differs from ID (e.g. a crate/formula/npm package name that is
	// not the same as the tool's binary or catalog ID). Empty means "use ID".
	PackageName string `yaml:"package_name,omitempty"`
	OfficialURL string `yaml:"official_url"`
	// Repo is the "org/repo" whose GitHub releases are this tool's upstream
	// release feed, for projects whose official_url points at a docs or
	// marketing site instead of the repository. Empty means "derive it from
	// official_url when that is itself a GitHub URL".
	Repo string `yaml:"repo,omitempty"`
	// VersionSource is an explicit authoritative "latest version" endpoint for
	// tools whose real release channel is neither a supported package-manager
	// registry nor a GitHub releases page (e.g. Go, whose downloads page is
	// not a repo). nil => fall back to registry/GitHub inference.
	VersionSource *VersionSourceSpec `yaml:"version_source,omitempty"`
}

// PkgName is the name to pass to a registry.Manager for this tool: the
// explicit PackageName override when set, otherwise the tool ID.
func (d ToolDef) PkgName() string {
	if d.PackageName != "" {
		return d.PackageName
	}
	return d.ID
}

// DetectSpec describes how to detect the installed version of a tool.
type DetectSpec struct {
	Cmd   string `yaml:"cmd"`
	Regex string `yaml:"regex"` // exactly one capture group, validated at load
}

// VersionSourceSpec points at a URL publishing a tool's current release, with
// a regex to pull the version out of the raw response body. Same one-capture-
// group contract as DetectSpec, so a plain-text endpoint, an HTML page, or a
// JSON document are all handled without a per-format parser.
type VersionSourceSpec struct {
	URL   string `yaml:"url"`
	Regex string `yaml:"regex"` // exactly one capture group, validated at load
}

// Category groups tools in the catalog and the TUI tree.
type Category struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// Profile is a named set of categories used to filter the TUI tree.
type Profile struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Categories []string `yaml:"categories"` // category IDs included in this profile
}
