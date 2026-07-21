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
	PackageName string      `yaml:"package_name,omitempty"`
	Update      *UpdateSpec `yaml:"update,omitempty"` // nil => no bespoke native updater
	OfficialURL string      `yaml:"official_url"`
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

// UpdateSpec is only present for tools with a bespoke native updater
// (e.g. `rustup update`, `copilot update`). Manager-owned tools get their
// update command synthesized at runtime by a registry.Manager.
type UpdateSpec struct {
	Cmd string `yaml:"cmd"`
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
