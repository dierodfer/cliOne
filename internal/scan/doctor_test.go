package scan

import (
	"testing"

	"github.com/dierodfer6/cliOne/internal/model"
)

func syntheticConflictCats() []model.CategoryState {
	tools := []model.ToolState{
		{
			Def:    model.ToolDef{ID: "git", Name: "Git"},
			Source: model.SourceResult{AllPaths: []string{"/usr/bin/git", "/usr/local/bin/git"}},
		},
		{
			Def:    model.ToolDef{ID: "ripgrep", Name: "ripgrep"},
			Source: model.SourceResult{AllPaths: []string{"/h/.cargo/bin/rg"}},
		},
	}
	return []model.CategoryState{
		{Category: model.Category{ID: "git", Name: "Git"}, Tools: tools},
	}
}

func TestConflictsFindsMultiPathTools(t *testing.T) {
	got := Conflicts(syntheticConflictCats())
	if len(got) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %+v", len(got), got)
	}
	if got[0].Tool.ID != "git" {
		t.Fatalf("expected git as the conflicting tool, got %q", got[0].Tool.ID)
	}
	if len(got[0].AllPaths) != 2 || got[0].AllPaths[0] != "/usr/bin/git" {
		t.Fatalf("expected active path first, got %v", got[0].AllPaths)
	}
}

func TestConflictsEmptyWhenNoOverlap(t *testing.T) {
	cats := []model.CategoryState{
		{Tools: []model.ToolState{
			{Def: model.ToolDef{ID: "x"}, Source: model.SourceResult{AllPaths: []string{"/only/path"}}},
		}},
	}
	if got := Conflicts(cats); len(got) != 0 {
		t.Fatalf("expected no conflicts, got %+v", got)
	}
}
