package model

import "testing"

func TestToolDefPkgName(t *testing.T) {
	if got := (ToolDef{ID: "ripgrep"}).PkgName(); got != "ripgrep" {
		t.Errorf("without override, PkgName should be the ID, got %q", got)
	}
	if got := (ToolDef{ID: "rg", PackageName: "ripgrep"}).PkgName(); got != "ripgrep" {
		t.Errorf("with override, PkgName should be PackageName, got %q", got)
	}
}
