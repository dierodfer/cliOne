package registry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCargoLatestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/crates/ripgrep" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"crate":{"max_stable_version":"14.1.0","newest_version":"14.1.0"}}`)
	}))
	defer srv.Close()

	c := &Cargo{BaseURL: srv.URL, Client: srv.Client()}
	v, err := c.LatestVersion(context.Background(), "ripgrep")
	if err != nil {
		t.Fatal(err)
	}
	if v != "14.1.0" {
		t.Fatalf("got %q, want 14.1.0", v)
	}
}

func TestCargoNotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	c := &Cargo{BaseURL: srv.URL, Client: srv.Client()}
	if _, err := c.LatestVersion(context.Background(), "ghost"); err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestNpmLatestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/npm/latest" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"name":"npm","version":"10.9.7"}`)
	}))
	defer srv.Close()

	n := &Npm{BaseURL: srv.URL, Client: srv.Client()}
	v, err := n.LatestVersion(context.Background(), "npm")
	if err != nil {
		t.Fatal(err)
	}
	if v != "10.9.7" {
		t.Fatalf("got %q, want 10.9.7", v)
	}
}

func TestPyPILatestVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pypi/httpie/json" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"info":{"name":"httpie","version":"3.2.4"}}`)
	}))
	defer srv.Close()

	p := &PyPI{BaseURL: srv.URL, Client: srv.Client()}
	v, err := p.LatestVersion(context.Background(), "httpie")
	if err != nil {
		t.Fatal(err)
	}
	if v != "3.2.4" {
		t.Fatalf("got %q, want 3.2.4", v)
	}
}

func TestGitHubReleaseLatestVersionViaRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/junegunn/fzf/releases/latest" {
			// A same-origin relative Location, built from a constant rather
			// than from request data: TagFromLocation only reads the path, and
			// echoing r.Host back would be a request-controlled redirect.
			w.Header().Set("Location", "/junegunn/fzf/releases/tag/v0.60.3")
			w.WriteHeader(http.StatusFound)
			return
		}
		// The client must NOT follow the redirect; landing here fails the test.
		http.Error(w, "redirect was followed", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := srv.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	g := &GitHubRelease{BaseURL: srv.URL, Client: client}
	v, err := g.LatestVersion(context.Background(), "junegunn/fzf")
	if err != nil {
		t.Fatal(err)
	}
	if v != "0.60.3" {
		t.Fatalf("got %q, want 0.60.3 (leading v stripped)", v)
	}
}

func TestGitHubReleaseNoRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	g := &GitHubRelease{BaseURL: srv.URL, Client: srv.Client()}
	if _, err := g.LatestVersion(context.Background(), "org/repo"); err == nil {
		t.Fatal("expected error when no redirect is returned")
	}
}

func TestTagFromLocation(t *testing.T) {
	cases := []struct {
		loc, want string
		wantErr   bool
	}{
		{"https://github.com/org/repo/releases/tag/v1.2.3", "1.2.3", false},
		{"https://github.com/org/repo/releases/tag/1.2.3?x=1", "1.2.3", false},
		{"https://github.com/org/repo/releases", "", true},
		{"https://github.com/org/repo/releases/tag/", "", true},
	}
	for _, tc := range cases {
		got, err := TagFromLocation(tc.loc)
		if tc.wantErr != (err != nil) {
			t.Fatalf("loc %q: err=%v, wantErr=%v", tc.loc, err, tc.wantErr)
		}
		if got != tc.want {
			t.Fatalf("loc %q: got %q, want %q", tc.loc, got, tc.want)
		}
	}
}

func TestBrewLatestVersionWithInjectedRunner(t *testing.T) {
	b := &Brew{Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "brew" {
			t.Fatalf("unexpected command %s", name)
		}
		return []byte(`{"formulae":[{"name":"jq","versions":{"stable":"1.7.1"}}]}`), nil
	}}
	v, err := b.LatestVersion(context.Background(), "jq")
	if err != nil {
		t.Fatal(err)
	}
	if v != "1.7.1" {
		t.Fatalf("got %q, want 1.7.1", v)
	}
}

func TestBrewLatestVersionFromCask(t *testing.T) {
	// GUI-app tools like vscode/temurin are distributed as Homebrew casks,
	// not formulae; `brew info --json=v2` reports these under "casks" with
	// the version given directly rather than nested under "versions".
	b := &Brew{Run: func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "brew" {
			t.Fatalf("unexpected command %s", name)
		}
		return []byte(`{"formulae":[],"casks":[{"token":"visual-studio-code","version":"1.85.1"}]}`), nil
	}}
	v, err := b.LatestVersion(context.Background(), "visual-studio-code")
	if err != nil {
		t.Fatal(err)
	}
	if v != "1.85.1" {
		t.Fatalf("got %q, want 1.85.1", v)
	}
}

func TestNormalizeVersionStripsProjectTagPrefixes(t *testing.T) {
	// Release tags carry project-specific prefixes. Left in place they break
	// numeric comparison, which reads as "up to date" instead of "outdated".
	cases := map[string]string{
		"v1.2.3":           "1.2.3",
		"1.2.3":            "1.2.3",
		"jq-1.8.2":         "1.8.2",
		"azure-cli-2.88.0": "2.88.0",
		"Helm v4.2.3":      "4.2.3",
		"  v0.60.3  ":      "0.60.3",
		"jdk-25.0.4+7":     "25.0.4",
		"no-digits-at-all": "no-digits-at-all",
	}
	for in, want := range cases {
		if got := normalizeVersion(in); got != want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDefaultRegistryCoversManagedKinds(t *testing.T) {
	r := Default()
	if len(r) != 4 {
		t.Fatalf("expected 4 managers, got %d", len(r))
	}
	for kind, m := range r {
		if m.Kind() != kind {
			t.Fatalf("manager registered under %v reports kind %v", kind, m.Kind())
		}
	}
}
