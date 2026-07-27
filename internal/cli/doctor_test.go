package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dierodfer6/cliOne/internal/cache"
	"github.com/dierodfer6/cliOne/internal/catalog"
	"github.com/dierodfer6/cliOne/internal/cli"
	"github.com/dierodfer6/cliOne/internal/model"
	"github.com/dierodfer6/cliOne/internal/registry"
	"github.com/dierodfer6/cliOne/internal/scan"
	"github.com/dierodfer6/cliOne/internal/source"
)

// mkExe creates an executable shell script at dir/name that prints version
// and returns its path.
func mkExe(t *testing.T, dir, name, version string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	script := "#!/bin/sh\necho " + version + "\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDoctorFindsPathConflicts(t *testing.T) {
	root := t.TempDir()
	d1 := filepath.Join(root, "one")
	d2 := filepath.Join(root, "two")
	mkExe(t, d1, "dup", "1.0.0")
	mkExe(t, d2, "dup", "1.0.0")
	t.Setenv("PATH", d1+string(os.PathListSeparator)+d2)

	cat := &catalog.Catalog{
		Categories: []model.Category{{ID: "utilities", Name: "Utilities"}},
		Tools: []model.ToolDef{{
			ID: "dup", Name: "Dup Tool", Category: "utilities",
			Detect:      model.DetectSpec{Cmd: "dup --version", Regex: `(\d+\.\d+\.\d+)`},
			OfficialURL: "https://example.com",
		}},
	}
	s := &scan.Scanner{
		Catalog:  cat,
		Cache:    cache.NewJSONStore(filepath.Join(t.TempDir(), "cache.json")),
		Resolver: source.NewResolver(),
		Registry: registry.Default(),
	}

	var buf bytes.Buffer
	code := cli.Doctor(context.Background(), s, &buf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	out := buf.String()
	if !strings.Contains(out, "Dup Tool") {
		t.Fatalf("expected the conflicting tool listed, got:\n%s", out)
	}
	if !strings.Contains(out, filepath.Join(d1, "dup")) || !strings.Contains(out, "active") {
		t.Fatalf("expected active path marked, got:\n%s", out)
	}
	if !strings.Contains(out, filepath.Join(d2, "dup")) || !strings.Contains(out, "shadowed") {
		t.Fatalf("expected shadowed path listed, got:\n%s", out)
	}
}

func TestDoctorNoConflicts(t *testing.T) {
	root := t.TempDir()
	mkExe(t, root, "solo", "1.0.0")
	t.Setenv("PATH", root)

	cat := &catalog.Catalog{
		Categories: []model.Category{{ID: "utilities", Name: "Utilities"}},
		Tools: []model.ToolDef{{
			ID: "solo", Name: "Solo Tool", Category: "utilities",
			Detect:      model.DetectSpec{Cmd: "solo --version", Regex: `(\d+\.\d+\.\d+)`},
			OfficialURL: "https://example.com",
		}},
	}
	s := &scan.Scanner{
		Catalog:  cat,
		Cache:    cache.NewJSONStore(filepath.Join(t.TempDir(), "cache.json")),
		Resolver: source.NewResolver(),
		Registry: registry.Default(),
	}

	var buf bytes.Buffer
	code := cli.Doctor(context.Background(), s, &buf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(buf.String(), "no PATH conflicts detected") {
		t.Fatalf("expected the no-conflicts message, got:\n%s", buf.String())
	}
}
