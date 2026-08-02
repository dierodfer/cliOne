package source

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dierodfer/cliOne/internal/model"
)

// mkExe creates an executable file at dir/name and returns its path.
func mkExe(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// resolveEval mirrors the production symlink resolution for expectations.
func resolveEval(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestScanPathOrderAndFiltering(t *testing.T) {
	root := t.TempDir()
	dirA := filepath.Join(root, "a")
	dirB := filepath.Join(root, "b")
	dirC := filepath.Join(root, "c")
	mkExe(t, dirA, "tool")
	mkExe(t, dirB, "tool")
	// dirC has a non-executable file with the right name: must be skipped.
	if err := os.MkdirAll(dirC, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirC, "tool"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	pathEnv := strings.Join([]string{dirB, dirC, dirA, filepath.Join(root, "missing")}, string(os.PathListSeparator))
	got := ScanPath("tool", pathEnv)
	want := []string{filepath.Join(dirB, "tool"), filepath.Join(dirA, "tool")}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestResolveClassifiesByPrefix(t *testing.T) {
	home := t.TempDir()
	cargoBin := filepath.Join(home, ".cargo", "bin")
	uvTools := filepath.Join(home, ".local", "share", "uv", "tools", "sometool", "bin")
	npmGlobal := filepath.Join(home, ".npm-global", "bin")
	nvmNodeBin := filepath.Join(home, ".nvm", "versions", "node", "v22.22.2", "bin")
	asdfShims := filepath.Join(home, ".asdf", "shims")
	asdfInstalls := filepath.Join(home, ".asdf", "installs", "ivm-node", "24.11.1", "bin")

	cases := []struct {
		name string
		dir  string
		want model.SourceKind
	}{
		{"cargo", cargoBin, model.SourceCargo},
		{"uv", uvTools, model.SourceUvTool},
		{"npm-global", npmGlobal, model.SourceNpmGlobal},
		// nvm installs each Node version under a variable version segment
		// (v22.22.2 above), which is exactly why global npm packages
		// installed via nvm used to be misclassified as SourceManual.
		{"nvm-npm-global", nvmNodeBin, model.SourceNpmGlobal},
		// asdf puts shims on $PATH; the shims are shell scripts rather than
		// symlinks, so classification has to recognize the shim directory
		// itself and never sees ~/.asdf/installs.
		{"asdf-shim", asdfShims, model.SourceAsdf},
		{"asdf-install", asdfInstalls, model.SourceAsdf},
		{"manual", filepath.Join(home, "custom", "bin"), model.SourceManual},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bin := mkExe(t, tc.dir, "mytool")
			r := &Resolver{
				Prefixes: DefaultPrefixes(resolveEval(t, home)),
				PathEnv:  tc.dir,
				DpkgOwns: func(context.Context, string) bool { return false },
			}
			res, err := r.Resolve(context.Background(), "mytool")
			if err != nil {
				t.Fatal(err)
			}
			if res.Kind != tc.want {
				t.Fatalf("got kind %v, want %v (binpath %s)", res.Kind, tc.want, res.BinPath)
			}
			if res.BinPath != resolveEval(t, bin) {
				t.Fatalf("got binpath %q, want %q", res.BinPath, bin)
			}
		})
	}
}

func TestResolveFollowsSymlinksToHomebrewCellar(t *testing.T) {
	// Simulate /usr/local/Cellar via a fake root: the prefix table is
	// injectable, so build one rooted in the temp dir.
	root := t.TempDir()
	cellar := filepath.Join(root, "Cellar", "jq", "1.7", "bin")
	real := mkExe(t, cellar, "jq")
	linkDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(linkDir, "jq")); err != nil {
		t.Fatal(err)
	}

	r := &Resolver{
		Prefixes: []Prefix{{Path: resolveEval(t, filepath.Join(root, "Cellar")) + "/", Kind: model.SourceHomebrew}},
		PathEnv:  linkDir,
		DpkgOwns: func(context.Context, string) bool { return false },
	}
	res, err := r.Resolve(context.Background(), "jq")
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != model.SourceHomebrew {
		t.Fatalf("got kind %v, want SourceHomebrew (binpath %s)", res.Kind, res.BinPath)
	}
	if res.BinPath != resolveEval(t, real) {
		t.Fatalf("symlink not resolved: got %q, want %q", res.BinPath, real)
	}
}

func TestResolveNotOnPath(t *testing.T) {
	r := &Resolver{
		Prefixes: DefaultPrefixes(t.TempDir()),
		PathEnv:  t.TempDir(),
		DpkgOwns: func(context.Context, string) bool { return false },
	}
	res, err := r.Resolve(context.Background(), "no-such-tool")
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != model.SourceUnknown || len(res.AllPaths) != 0 {
		t.Fatalf("expected SourceUnknown with no paths, got %+v", res)
	}
}

func TestClassifyAmbiguousUsrBinUsesDpkg(t *testing.T) {
	// Pure classification: /usr/bin paths stay Manual unless dpkg confirms.
	prefixes := DefaultPrefixes("/home/nobody")
	if got := ClassifyPath("/usr/bin/git", prefixes); got != model.SourceManual {
		t.Fatalf("pre-dpkg classification of /usr/bin should be Manual, got %v", got)
	}
	if !isAmbiguousSystemPath("/usr/bin/git") {
		t.Fatal("/usr/bin/git should be ambiguous")
	}
	if isAmbiguousSystemPath("/opt/thing/git") {
		t.Fatal("/opt/thing/git should not be ambiguous")
	}
}

func TestResolveDpkgPromotion(t *testing.T) {
	// A binary whose resolved path is under /usr/bin cannot be fabricated in
	// a temp dir, so exercise the promotion logic through the injectable
	// DpkgOwns hook plus a fake ambiguity: use the real /usr/bin only if a
	// known binary exists there.
	if _, err := os.Stat("/usr/bin/env"); err != nil {
		t.Skip("no /usr/bin/env on this system")
	}
	r := &Resolver{
		Prefixes: DefaultPrefixes("/home/nobody"),
		PathEnv:  "/usr/bin",
		DpkgOwns: func(_ context.Context, path string) bool { return path == "/usr/bin/env" },
	}
	res, err := r.Resolve(context.Background(), "env")
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != model.SourceAptDnf {
		t.Fatalf("expected SourceAptDnf after dpkg confirmation, got %v (binpath %s)", res.Kind, res.BinPath)
	}
}

func TestScanPathMultipleLocationsFeedAllPaths(t *testing.T) {
	root := t.TempDir()
	d1 := filepath.Join(root, "one")
	d2 := filepath.Join(root, "two")
	mkExe(t, d1, "dup")
	mkExe(t, d2, "dup")
	r := &Resolver{
		Prefixes: DefaultPrefixes(root),
		PathEnv:  d1 + string(os.PathListSeparator) + d2,
		DpkgOwns: func(context.Context, string) bool { return false },
	}
	res, err := r.Resolve(context.Background(), "dup")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.AllPaths) != 2 {
		t.Fatalf("expected 2 paths for doctor view, got %v", res.AllPaths)
	}
	if res.AllPaths[0] != filepath.Join(d1, "dup") {
		t.Fatalf("active path should be first on PATH, got %v", res.AllPaths)
	}
}
