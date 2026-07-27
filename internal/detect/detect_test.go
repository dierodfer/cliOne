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
	"git":            "2.43.0",
	"fzf":            "0.60.3",
	"ripgrep":        "14.1.0",
	"bat":            "0.24.0",
	"jq":             "1.7",
	"yq":             "4.40.5",
	"homebrew":       "4.2.21",
	"rustup":         "1.94.1",
	"node":           "22.22.2",
	"go":             "1.24.7",
	"python":         "3.11.4",
	"java":           "17.0.8",
	"dotnet":         "8.0.100",
	"npm":            "10.9.7",
	"pnpm":           "8.6.10",
	"yarn":           "1.22.19",
	"poetry":         "1.7.1",
	"gh":             "2.45.0",
	"lazygit":        "0.40.2",
	"copilot":        "1.0.0",
	"claude-code":    "2.1.216",
	"opencode":       "0.1.5",
	"codex":          "0.5.0",
	"gemini":         "0.2.1",
	"aider":          "0.60.0",
	"kubectx":        "0.9.5",
	"helm":           "3.13.2",
	"k9s":            "0.29.1",
	"minikube":       "1.32.0",
	"docker":         "29.3.1",
	"docker-compose": "2.23.0",
	"podman":         "4.7.2",
	"aws-cli":        "2.15.0",
	"azure-cli":      "2.55.0",
	"gcloud":         "455.0.0",
	"terraform":      "1.6.6",
	"vault":          "1.15.4",
	"vscode":         "1.85.1",
	"neovim":         "0.9.4",
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
