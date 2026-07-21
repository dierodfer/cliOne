package updater

import (
	"testing"

	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/registry"
)

func def(withUpdate bool) model.ToolDef {
	d := model.ToolDef{ID: "tool", OfficialURL: "https://example.com"}
	if withUpdate {
		d.Update = &model.UpdateSpec{Cmd: "tool self-update"}
	}
	return d
}

func src(k model.SourceKind) model.SourceResult {
	return model.SourceResult{Kind: k, BinPath: "/x/tool"}
}

func TestDecideRoutingTable(t *testing.T) {
	reg := registry.Default()
	cases := []struct {
		name      string
		hasUpdate bool
		kind      model.SourceKind
		status    model.StatusState
		want      model.UpdateAction
	}{
		{"not installed", false, model.SourceUnknown, model.StatusNotInstalled, model.ActionOpenOfficialPage},
		{"not installed with update spec", true, model.SourceUnknown, model.StatusNotInstalled, model.ActionOpenOfficialPage},
		{"latest unknown", false, model.SourceManual, model.StatusLatestUnknown, model.ActionOpenOfficialPage},
		{"up to date", true, model.SourceHomebrew, model.StatusUpToDate, model.ActionNone},
		{"update avail, native updater wins", true, model.SourceHomebrew, model.StatusUpdateAvail, model.ActionRunNativeUpdate},
		{"update avail, homebrew owned", false, model.SourceHomebrew, model.StatusUpdateAvail, model.ActionRunManagerUpdate},
		{"update avail, cargo owned", false, model.SourceCargo, model.StatusUpdateAvail, model.ActionRunManagerUpdate},
		{"update avail, npm owned", false, model.SourceNpmGlobal, model.StatusUpdateAvail, model.ActionRunManagerUpdate},
		{"update avail, uv owned", false, model.SourceUvTool, model.StatusUpdateAvail, model.ActionRunManagerUpdate},
		{"update avail, manual install", false, model.SourceManual, model.StatusUpdateAvail, model.ActionOpenOfficialPage},
		{"update avail, apt owned", false, model.SourceAptDnf, model.StatusUpdateAvail, model.ActionOpenOfficialPage},
		{"update avail, unknown source", false, model.SourceUnknown, model.StatusUpdateAvail, model.ActionOpenOfficialPage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(def(tc.hasUpdate), src(tc.kind), tc.status, reg)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCommandSynthesis(t *testing.T) {
	reg := registry.Default()

	native := Command(def(true), src(model.SourceManual), model.ActionRunNativeUpdate, reg)
	if len(native) != 2 || native[0] != "tool" || native[1] != "self-update" {
		t.Fatalf("unexpected native command %v", native)
	}

	brew := Command(def(false), src(model.SourceHomebrew), model.ActionRunManagerUpdate, reg)
	if len(brew) != 3 || brew[0] != "brew" || brew[1] != "upgrade" || brew[2] != "tool" {
		t.Fatalf("unexpected brew command %v", brew)
	}

	cargo := Command(def(false), src(model.SourceCargo), model.ActionRunManagerUpdate, reg)
	if len(cargo) != 4 || cargo[0] != "cargo" || cargo[3] != "--force" {
		t.Fatalf("unexpected cargo command %v", cargo)
	}

	if got := Command(def(false), src(model.SourceManual), model.ActionOpenOfficialPage, reg); got != nil {
		t.Fatalf("open-page action must have no command, got %v", got)
	}
	if got := Command(def(false), src(model.SourceManual), model.ActionRunManagerUpdate, reg); got != nil {
		t.Fatalf("manager update without a manager must be nil, got %v", got)
	}

	// A PackageName override is what the manager command targets, not the ID.
	renamed := model.ToolDef{ID: "tool", PackageName: "the-crate", OfficialURL: "https://example.com"}
	got := Command(renamed, src(model.SourceCargo), model.ActionRunManagerUpdate, reg)
	if len(got) != 4 || got[2] != "the-crate" {
		t.Fatalf("expected cargo command to target package override, got %v", got)
	}
}
