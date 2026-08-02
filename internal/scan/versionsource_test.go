package scan

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dierodfer/cliOne/internal/model"
	"github.com/dierodfer/cliOne/internal/registry"
)

func TestFetchVersionSourceExtractsFromPlainText(t *testing.T) {
	// Shaped like the real go.dev/VERSION?m=text response: the version on the
	// first line, a build timestamp on the second that must not be picked up.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "go1.24.7\ntime 2025-08-07T15:04:05Z\n")
	}))
	defer srv.Close()

	got, err := fetchVersionSource(context.Background(), model.VersionSourceSpec{
		URL:   srv.URL,
		Regex: `go(\d+\.\d+(?:\.\d+)?)`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.24.7" {
		t.Fatalf("got %q, want 1.24.7", got)
	}
}

func TestFetchVersionSourceSendsUserAgent(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("User-Agent")
		fmt.Fprint(w, "go1.24.7\n")
	}))
	defer srv.Close()

	if _, err := fetchVersionSource(context.Background(), model.VersionSourceSpec{
		URL:   srv.URL,
		Regex: `go(\d+\.\d+(?:\.\d+)?)`,
	}); err != nil {
		t.Fatal(err)
	}
	if seen != versionSourceUserAgent {
		t.Fatalf("got User-Agent %q, want %q", seen, versionSourceUserAgent)
	}
}

func TestFetchVersionSourceErrors(t *testing.T) {
	okBody := func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "go1.24.7\n") }

	cases := []struct {
		name    string
		handler http.HandlerFunc
		regex   string
		wantErr string
	}{
		{
			name:    "non-200 status",
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) },
			regex:   `go(\d+\.\d+(?:\.\d+)?)`,
			wantErr: "HTTP 503",
		},
		{
			name:    "no match in body",
			handler: func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "under maintenance\n") },
			regex:   `go(\d+\.\d+(?:\.\d+)?)`,
			wantErr: "no version match",
		},
		{
			name:    "regex does not compile",
			handler: okBody,
			regex:   `go(\d+`,
			wantErr: "regex does not compile",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			_, err := fetchVersionSource(context.Background(), model.VersionSourceSpec{URL: srv.URL, Regex: tc.regex})
			if err == nil {
				t.Fatalf("expected an error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got error %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestFetchLatestPrefersVersionSourceOverRegistry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "go1.26.5\n")
	}))
	defer srv.Close()

	// A tool that is Homebrew-owned (so the registry branch would normally win)
	// but declares a version_source: the explicit declaration must take
	// precedence over manager inference.
	s := &Scanner{Cache: noopCache{}}
	def := model.ToolDef{
		ID:          "go",
		OfficialURL: "https://go.dev/dl/",
		VersionSource: &model.VersionSourceSpec{
			URL:   srv.URL,
			Regex: `go(\d+\.\d+(?:\.\d+)?)`,
		},
	}
	got := s.FetchLatest(context.Background(), def, model.SourceResult{Kind: model.SourceHomebrew})
	if got.Result.Err != nil {
		t.Fatal(got.Result.Err)
	}
	if got.Result.Latest != "1.26.5" {
		t.Fatalf("got latest %q, want 1.26.5", got.Result.Latest)
	}
}

func TestFetchLatestPrefersGitHubOverManagerRegistry(t *testing.T) {
	// A brew-installed tool whose upstream is GitHub must report the upstream
	// release, not what the manager happens to distribute — a manager that
	// trails upstream is exactly what makes an outdated tool look current.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/helm/helm/releases/latest" {
			w.Header().Set("Location", "/helm/helm/releases/tag/v4.2.3")
			w.WriteHeader(http.StatusFound)
			return
		}
		http.Error(w, "unexpected path", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := srv.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	s := &Scanner{
		Cache:    noopCache{},
		Registry: registry.Registry{model.SourceHomebrew: stubManager{version: "3.13.2"}},
		GitHub:   &registry.GitHubRelease{BaseURL: srv.URL, Client: client},
	}
	def := model.ToolDef{ID: "helm", Repo: "helm/helm", OfficialURL: "https://helm.sh"}

	got := s.FetchLatest(context.Background(), def, model.SourceResult{Kind: model.SourceHomebrew})
	if got.Result.Err != nil {
		t.Fatal(got.Result.Err)
	}
	if got.Result.Latest != "4.2.3" {
		t.Fatalf("got latest %q from the manager, want 4.2.3 from upstream", got.Result.Latest)
	}
}

func TestFetchLatestFallsBackToManagerRegistry(t *testing.T) {
	// With no upstream source declared and no GitHub official URL, the owning
	// manager is still better than reporting nothing.
	s := &Scanner{
		Cache:    noopCache{},
		Registry: registry.Registry{model.SourceCargo: stubManager{version: "14.1.0"}},
	}
	def := model.ToolDef{ID: "ripgrep", OfficialURL: "https://example.com/ripgrep"}

	got := s.FetchLatest(context.Background(), def, model.SourceResult{Kind: model.SourceCargo})
	if got.Result.Err != nil {
		t.Fatal(got.Result.Err)
	}
	if got.Result.Latest != "14.1.0" {
		t.Fatalf("got latest %q, want 14.1.0 from the manager fallback", got.Result.Latest)
	}
}

func TestFetchLatestWithoutAnySourceReportsError(t *testing.T) {
	s := &Scanner{Cache: noopCache{}, Registry: registry.Registry{}}
	def := model.ToolDef{ID: "orphan", OfficialURL: "https://example.com"}

	got := s.FetchLatest(context.Background(), def, model.SourceResult{Kind: model.SourceAsdf})
	if !errors.Is(got.Result.Err, ErrNoLatestSource) {
		t.Fatalf("got err %v, want ErrNoLatestSource", got.Result.Err)
	}
}

// stubManager is a registry.Manager returning a fixed version.
type stubManager struct{ version string }

func (stubManager) Kind() model.SourceKind { return model.SourceUnknown }
func (m stubManager) LatestVersion(context.Context, string) (string, error) {
	return m.version, nil
}

// noopCache satisfies cache.Cache without touching disk.
type noopCache struct{}

func (noopCache) GetSource(string) (model.SourceResult, bool)  { return model.SourceResult{}, false }
func (noopCache) SetSource(string, model.SourceResult) error   { return nil }
func (noopCache) GetLatest(string) (model.VersionResult, bool) { return model.VersionResult{}, false }
func (noopCache) SetLatest(string, model.VersionResult) error  { return nil }
