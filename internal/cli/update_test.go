package cli_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
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

func TestUpdateUnknownTool(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	s := testScanner(t, cat)

	var stdout, stderr bytes.Buffer
	code := cli.Update(context.Background(), s, &stdout, &stderr, "not-a-real-tool-id")
	if code != cli.ExitUnknownTool {
		t.Fatalf("expected exit %d, got %d", cli.ExitUnknownTool, code)
	}
	if !strings.Contains(stderr.String(), "not-a-real-tool-id") {
		t.Fatalf("expected the unknown ID in stderr, got %q", stderr.String())
	}
}

func TestUpdateNotInstalledOpensOfficialPage(t *testing.T) {
	cat := &catalog.Catalog{
		Categories: []model.Category{{ID: "utilities", Name: "Utilities"}},
		Tools: []model.ToolDef{{
			ID: "ghost", Name: "Ghost Tool", Category: "utilities",
			Detect:      model.DetectSpec{Cmd: "definitely-not-a-real-binary-xyz --version", Regex: `(\d+\.\d+\.\d+)`},
			OfficialURL: "https://example.com/ghost",
		}},
	}
	s := testScanner(t, cat)

	var stdout, stderr bytes.Buffer
	code := cli.Update(context.Background(), s, &stdout, &stderr, "ghost")
	if code != cli.ExitNoUpdater {
		t.Fatalf("expected exit %d, got %d; stderr=%s", cli.ExitNoUpdater, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "https://example.com/ghost") {
		t.Fatalf("expected the official URL in stdout, got %q", stdout.String())
	}
}

// githubRedirectServer fakes the GitHub /releases/latest redirect trick so
// tests can reach an "update available" status without live network.
func githubRedirectServer(t *testing.T, tag string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/testorg/testrepo/releases/tag/"+tag)
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// noRedirectClient mirrors registry.NewGitHubRelease's real client: it must
// capture the redirect response instead of following it, exactly like
// production does against real github.com.
func noRedirectClient(srv *httptest.Server) *http.Client {
	c := *srv.Client()
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &c
}

func TestUpdateRunsNativeUpdaterAndReportsSuccess(t *testing.T) {
	srv := githubRedirectServer(t, "v9.9.9")

	root := t.TempDir()
	mkExe(t, root, "dup", "1.0.0")
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	cat := &catalog.Catalog{
		Categories: []model.Category{{ID: "utilities", Name: "Utilities"}},
		Tools: []model.ToolDef{{
			ID: "dup-ok", Name: "Dup OK", Category: "utilities",
			Detect:      model.DetectSpec{Cmd: "dup --version", Regex: `(\d+\.\d+\.\d+)`},
			Update:      &model.UpdateSpec{Cmd: "true"},
			OfficialURL: "https://github.com/testorg/testrepo",
		}},
	}
	s := &scan.Scanner{
		Catalog:  cat,
		Cache:    cache.NewJSONStore(filepath.Join(t.TempDir(), "cache.json")),
		Resolver: source.NewResolver(),
		Registry: registry.Default(),
		GitHub:   &registry.GitHubRelease{BaseURL: srv.URL, Client: noRedirectClient(srv)},
	}

	var stdout, stderr bytes.Buffer
	code := cli.Update(context.Background(), s, &stdout, &stderr, "dup-ok")
	if code != cli.ExitOK {
		t.Fatalf("expected exit %d, got %d; stderr=%s", cli.ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "updated successfully") {
		t.Fatalf("expected success message, got %q", stdout.String())
	}
}

func TestUpdateReportsFailure(t *testing.T) {
	srv := githubRedirectServer(t, "v9.9.9")

	root := t.TempDir()
	mkExe(t, root, "dup", "1.0.0")
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	cat := &catalog.Catalog{
		Categories: []model.Category{{ID: "utilities", Name: "Utilities"}},
		Tools: []model.ToolDef{{
			ID: "dup-fail", Name: "Dup Fail", Category: "utilities",
			Detect:      model.DetectSpec{Cmd: "dup --version", Regex: `(\d+\.\d+\.\d+)`},
			Update:      &model.UpdateSpec{Cmd: "false"},
			OfficialURL: "https://github.com/testorg/testrepo",
		}},
	}
	s := &scan.Scanner{
		Catalog:  cat,
		Cache:    cache.NewJSONStore(filepath.Join(t.TempDir(), "cache.json")),
		Resolver: source.NewResolver(),
		Registry: registry.Default(),
		GitHub:   &registry.GitHubRelease{BaseURL: srv.URL, Client: noRedirectClient(srv)},
	}

	var stdout, stderr bytes.Buffer
	code := cli.Update(context.Background(), s, &stdout, &stderr, "dup-fail")
	if code != cli.ExitUpdateFailed {
		t.Fatalf("expected exit %d, got %d", cli.ExitUpdateFailed, code)
	}
	if !strings.Contains(stderr.String(), "failed") {
		t.Fatalf("expected a failure message in stderr, got %q", stderr.String())
	}
}

func TestUpdateAlreadyUpToDate(t *testing.T) {
	srv := githubRedirectServer(t, "v1.0.0")

	root := t.TempDir()
	mkExe(t, root, "dup", "1.0.0")
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

	cat := &catalog.Catalog{
		Categories: []model.Category{{ID: "utilities", Name: "Utilities"}},
		Tools: []model.ToolDef{{
			ID: "dup-current", Name: "Dup Current", Category: "utilities",
			Detect:      model.DetectSpec{Cmd: "dup --version", Regex: `(\d+\.\d+\.\d+)`},
			OfficialURL: "https://github.com/testorg/testrepo",
		}},
	}
	s := &scan.Scanner{
		Catalog:  cat,
		Cache:    cache.NewJSONStore(filepath.Join(t.TempDir(), "cache.json")),
		Resolver: source.NewResolver(),
		Registry: registry.Default(),
		GitHub:   &registry.GitHubRelease{BaseURL: srv.URL, Client: noRedirectClient(srv)},
	}

	var stdout, stderr bytes.Buffer
	code := cli.Update(context.Background(), s, &stdout, &stderr, "dup-current")
	if code != cli.ExitOK {
		t.Fatalf("expected exit %d, got %d; stderr=%s", cli.ExitOK, code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "already up to date") {
		t.Fatalf("expected up-to-date message, got %q", stdout.String())
	}
}
