package catalog

import (
	"strings"
	"testing"
)

func TestLoadEmbeddedCatalog(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Categories) != 10 {
		t.Errorf("got %d categories, want 10", len(c.Categories))
	}
	if len(c.Profiles) != 5 {
		t.Errorf("got %d profiles, want 5", len(c.Profiles))
	}
	if len(c.Tools) != 39 {
		t.Errorf("got %d tools, want 39", len(c.Tools))
	}
	if _, ok := c.CategoryByID("utilities"); !ok {
		t.Errorf("expected category utilities to exist")
	}
	if got := len(c.ToolsByCategory("git")); got != 3 {
		t.Errorf("got %d tools in git category, want 3 (git, gh, lazygit)", got)
	}
}

func TestUpdateSpecPresence(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	withUpdate := map[string]bool{"homebrew": true, "rustup": true, "npm": true, "copilot": true}
	for _, tool := range c.Tools {
		if withUpdate[tool.ID] && tool.Update == nil {
			t.Errorf("tool %q should have an update spec", tool.ID)
		}
		if !withUpdate[tool.ID] && tool.Update != nil {
			t.Errorf("tool %q should not have an update spec", tool.ID)
		}
	}
}

const validHeader = `
categories:
  - {id: c1, name: One}
profiles:
  - {id: p1, name: P, categories: [c1]}
`

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "regex with no capture group",
			yaml: validHeader + `
tools:
  - id: t1
    name: T1
    category: c1
    detect: {cmd: "t1 --version", regex: '\d+\.\d+'}
    official_url: "https://example.com"
`,
			wantErr: `tool "t1" detect.regex must have exactly one capture group`,
		},
		{
			name: "regex with two capture groups",
			yaml: validHeader + `
tools:
  - id: t1
    name: T1
    category: c1
    detect: {cmd: "t1 --version", regex: '(\d+)\.(\d+)'}
    official_url: "https://example.com"
`,
			wantErr: `exactly one capture group, has 2`,
		},
		{
			name: "regex does not compile",
			yaml: validHeader + `
tools:
  - id: t1
    name: T1
    category: c1
    detect: {cmd: "t1 --version", regex: '(unclosed'}
    official_url: "https://example.com"
`,
			wantErr: `tool "t1" detect.regex does not compile`,
		},
		{
			name: "unknown category reference",
			yaml: validHeader + `
tools:
  - id: t1
    name: T1
    category: nope
    detect: {cmd: "t1 --version", regex: '(\d+)'}
    official_url: "https://example.com"
`,
			wantErr: `tool "t1" references unknown category "nope"`,
		},
		{
			name: "profile references unknown category",
			yaml: `
categories:
  - {id: c1, name: One}
profiles:
  - {id: p1, name: P, categories: [ghost]}
tools: []
`,
			wantErr: `profile "p1" references unknown category "ghost"`,
		},
		{
			name: "missing official_url",
			yaml: validHeader + `
tools:
  - id: t1
    name: T1
    category: c1
    detect: {cmd: "t1 --version", regex: '(\d+)'}
`,
			wantErr: `tool "t1" is missing official_url`,
		},
		{
			name: "duplicate tool ids",
			yaml: validHeader + `
tools:
  - id: t1
    name: T1
    category: c1
    detect: {cmd: "t1 --version", regex: '(\d+)'}
    official_url: "https://example.com"
  - id: t1
    name: T1 again
    category: c1
    detect: {cmd: "t1 --version", regex: '(\d+)'}
    official_url: "https://example.com"
`,
			wantErr: `duplicate tool id "t1"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}
