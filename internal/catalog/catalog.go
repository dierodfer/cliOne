// Package catalog loads and validates the embedded tool catalog.
package catalog

import (
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/dierodfer/cliOne/internal/model"
)

//go:embed data/tools.yaml
var dataFS embed.FS

// Catalog is the parsed and validated tool catalog.
type Catalog struct {
	Categories []model.Category `yaml:"categories"`
	Profiles   []model.Profile  `yaml:"profiles"`
	Tools      []model.ToolDef  `yaml:"tools"`
}

// Load parses the embedded catalog YAML and validates it, failing fast on any
// schema violation.
func Load() (*Catalog, error) {
	raw, err := dataFS.ReadFile("data/tools.yaml")
	if err != nil {
		return nil, fmt.Errorf("catalog: reading embedded data: %w", err)
	}
	return Parse(raw)
}

// Parse parses and validates catalog YAML bytes. Exposed for tests.
func Parse(raw []byte) (*Catalog, error) {
	var c Catalog
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("catalog: parsing YAML: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// CategoryByID returns the category with the given ID.
func (c *Catalog) CategoryByID(id string) (model.Category, bool) {
	for _, cat := range c.Categories {
		if cat.ID == id {
			return cat, true
		}
	}
	return model.Category{}, false
}

// ToolsByCategory returns the tools belonging to the given category ID, in
// catalog order.
func (c *Catalog) ToolsByCategory(catID string) []model.ToolDef {
	var out []model.ToolDef
	for _, t := range c.Tools {
		if t.Category == catID {
			out = append(out, t)
		}
	}
	return out
}

// ToolByID returns the tool with the given catalog ID.
func (c *Catalog) ToolByID(id string) (model.ToolDef, bool) {
	for _, t := range c.Tools {
		if t.ID == id {
			return t, true
		}
	}
	return model.ToolDef{}, false
}
