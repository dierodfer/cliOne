package scan

import (
	"testing"

	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/registry"
)

func TestComputeStatus(t *testing.T) {
	reg := registry.Default()
	withUpdate := model.ToolDef{ID: "t", Update: &model.UpdateSpec{Cmd: "t update"}}
	plain := model.ToolDef{ID: "t"}
	installed := func(v string) model.DetectResult { return model.DetectResult{Installed: true, Version: v} }

	cases := []struct {
		name   string
		def    model.ToolDef
		det    model.DetectResult
		src    model.SourceResult
		latest model.VersionResult
		want   model.StatusState
	}{
		{"not installed", plain, model.DetectResult{}, model.SourceResult{}, model.VersionResult{}, model.StatusNotInstalled},
		{"installed manual no updater", plain, installed("1.0.0"), model.SourceResult{Kind: model.SourceManual}, model.VersionResult{}, model.StatusNoUpdater},
		{"installed apt no updater", plain, installed("1.0.0"), model.SourceResult{Kind: model.SourceAptDnf}, model.VersionResult{}, model.StatusNoUpdater},
		{"native updater latest unknown", withUpdate, installed("1.0.0"), model.SourceResult{Kind: model.SourceManual}, model.VersionResult{}, model.StatusUpToDate},
		{"native updater newer available", withUpdate, installed("1.0.0"), model.SourceResult{Kind: model.SourceManual}, model.VersionResult{Latest: "1.1.0"}, model.StatusUpdateAvail},
		{"native updater current", withUpdate, installed("1.1.0"), model.SourceResult{Kind: model.SourceManual}, model.VersionResult{Latest: "1.1.0"}, model.StatusUpToDate},
		{"brew owned newer available", plain, installed("1.0.0"), model.SourceResult{Kind: model.SourceHomebrew}, model.VersionResult{Latest: "2.0.0"}, model.StatusUpdateAvail},
		{"cargo owned current", plain, installed("14.1.0"), model.SourceResult{Kind: model.SourceCargo}, model.VersionResult{Latest: "14.1.0"}, model.StatusUpToDate},
		{"npm owned latest unknown", plain, installed("1.0.0"), model.SourceResult{Kind: model.SourceNpmGlobal}, model.VersionResult{}, model.StatusUpToDate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeStatus(tc.def, tc.det, tc.src, tc.latest, reg)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBinName(t *testing.T) {
	def := model.ToolDef{ID: "ripgrep", Detect: model.DetectSpec{Cmd: "rg --version"}}
	if got := BinName(def); got != "rg" {
		t.Fatalf("got %q, want rg", got)
	}
	if got := BinName(model.ToolDef{ID: "x"}); got != "x" {
		t.Fatalf("empty cmd should fall back to ID, got %q", got)
	}
}

func TestGithubRepoFromURL(t *testing.T) {
	cases := []struct {
		url  string
		want string
		ok   bool
	}{
		{"https://github.com/junegunn/fzf", "junegunn/fzf", true},
		{"https://github.com/github/copilot-cli", "github/copilot-cli", true},
		{"https://github.com/org/repo/releases", "org/repo", true},
		{"https://nodejs.org", "", false},
		{"https://github.com/onlyorg", "", false},
	}
	for _, tc := range cases {
		got, ok := githubRepoFromURL(tc.url)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("url %q: got (%q,%v), want (%q,%v)", tc.url, got, ok, tc.want, tc.ok)
		}
	}
}
