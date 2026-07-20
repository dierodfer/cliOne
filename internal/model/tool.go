// Package model holds CLIOne's shared domain types.
//
// It is the vocabulary shared by every other internal package and must not
// import any other internal package.
package model

// ToolDef describes a tool tracked by CLIOne, as declared in the catalog.
type ToolDef struct {
	ID          string      `yaml:"id"`
	Name        string      `yaml:"name"`
	Category    string      `yaml:"category"`
	Detect      DetectSpec  `yaml:"detect"`
	Update      *UpdateSpec `yaml:"update,omitempty"` // nil => no bespoke native updater
	OfficialURL string      `yaml:"official_url"`
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
