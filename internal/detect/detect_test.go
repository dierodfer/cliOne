package detect

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/dierodfer6/cliOne/internal/catalog"
	"github.com/dierodfer6/cliOne/internal/model"
)

// fixtureVersions maps tool ID -> version expected from testdata/detect/<id>.txt.
var fixtureVersions = map[string]string{
	"git":      "2.43.0",
	"fzf":      "0.60.3",
	"ripgrep":  "14.1.0",
	"bat":      "0.24.0",
	"jq":       "1.7",
	"homebrew": "4.2.21",
	"rustup":   "1.94.1",
	"node":     "22.22.2",
	"npm":      "10.9.7",
	"gh":       "2.45.0",
	"copilot":  "1.0.0",
	"kubectx":  "0.9.5",
	"lazygit":  "0.40.2",
}

func TestExtractVersionAgainstFixtures(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("loading catalog: %v", err)
	}
	for _, def := range cat.Tools {
		def := def
		t.Run(def.ID, func(t *testing.T) {
			want, ok := fixtureVersions[def.ID]
			if !ok {
				t.Fatalf("no fixture expectation for catalog tool %q; add testdata/detect/%s.txt and an entry here", def.ID, def.ID)
			}
			raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "detect", def.ID+".txt"))
			if err != nil {
				t.Fatalf("reading fixture: %v", err)
			}
			re := regexp.MustCompile(def.Detect.Regex)
			res := ExtractVersion(re, string(raw))
			if !res.Installed {
				t.Fatalf("regex %q did not match fixture output %q", def.Detect.Regex, raw)
			}
			if res.Version != want {
				t.Fatalf("got version %q, want %q", res.Version, want)
			}
		})
	}
}

func TestExtractVersionNoMatch(t *testing.T) {
	re := regexp.MustCompile(`git version (\d+\.\d+\.\d+)`)
	res := ExtractVersion(re, "command not found")
	if res.Installed || res.Version != "" {
		t.Fatalf("expected no match, got %+v", res)
	}
}

func TestRunDetectMissingBinary(t *testing.T) {
	def := model.ToolDef{
		ID:     "ghost",
		Detect: model.DetectSpec{Cmd: "definitely-not-a-real-binary-xyz --version", Regex: `(\d+)`},
	}
	res, err := RunDetect(context.Background(), def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Installed {
		t.Fatalf("expected Installed=false for missing binary, got %+v", res)
	}
}

func TestRunDetectRealCommand(t *testing.T) {
	// `go version` is guaranteed present in the test environment.
	def := model.ToolDef{
		ID:     "go",
		Detect: model.DetectSpec{Cmd: "go version", Regex: `go(\d+\.\d+(?:\.\d+)?)`},
	}
	res, err := RunDetect(context.Background(), def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Installed || res.Version == "" {
		t.Fatalf("expected installed go with a version, got %+v", res)
	}
}

func TestRunDetectUnparsableOutput(t *testing.T) {
	def := model.ToolDef{
		ID:     "true",
		Detect: model.DetectSpec{Cmd: "true", Regex: `(\d+\.\d+\.\d+)`},
	}
	res, err := RunDetect(context.Background(), def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Installed {
		t.Fatalf("binary on PATH should count as installed, got %+v", res)
	}
	if res.Err == nil {
		t.Fatalf("expected a parse error when output has no version")
	}
}
