package cli_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dierodfer/cliOne/internal/cache"
	"github.com/dierodfer/cliOne/internal/catalog"
	"github.com/dierodfer/cliOne/internal/cli"
	"github.com/dierodfer/cliOne/internal/registry"
	"github.com/dierodfer/cliOne/internal/scan"
	"github.com/dierodfer/cliOne/internal/source"
)

func testScanner(t *testing.T, cat *catalog.Catalog) *scan.Scanner {
	t.Helper()
	return &scan.Scanner{
		Catalog:  cat,
		Cache:    cache.NewJSONStore(filepath.Join(t.TempDir(), "cache.json")),
		Resolver: source.NewResolver(),
		Registry: registry.Default(),
	}
}

func TestListPrintsEveryCategoryAndTool(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	s := testScanner(t, cat)

	var buf bytes.Buffer
	code := cli.List(context.Background(), s, &buf, false)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	out := buf.String()
	for _, cs := range cat.Categories {
		if !strings.Contains(out, cs.Name) {
			t.Errorf("expected category %q in output", cs.Name)
		}
	}
	// A tool with a well-known ID unlikely to be installed on a CI/dev box.
	if !strings.Contains(out, "minikube") {
		t.Errorf("expected tool id %q in output, got:\n%s", "minikube", out)
	}
}
