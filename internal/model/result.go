package model

import "time"

// DetectResult is the outcome of running a tool's detect command.
type DetectResult struct {
	Installed bool
	Version   string
	RawOutput string
	Err       error
}

// SourceResult is the outcome of resolving which manager owns a binary.
type SourceResult struct {
	Kind     SourceKind
	BinPath  string   // resolved path of the active binary (first on $PATH)
	AllPaths []string // every resolvable location for this binary on $PATH
}

// VersionResult is the outcome of a latest-version lookup.
type VersionResult struct {
	Latest    string
	Err       error
	FetchedAt time.Time
}

// ToolState is everything CLIOne knows about one tool at render time.
type ToolState struct {
	Def    ToolDef
	Status StatusState
	Detect DetectResult
	Source SourceResult
	Latest VersionResult
}

// CategoryState groups tool states under a category with installed/total counts.
type CategoryState struct {
	Category  Category
	Tools     []ToolState
	Installed int
	Total     int
}
