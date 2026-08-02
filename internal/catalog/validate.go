package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dierodfer/cliOne/internal/model"
)

// validate enforces the catalog schema rules, naming the offending tool ID in
// every error. The per-section rules live in the helpers below so each one
// stays readable on its own.
func (c *Catalog) validate() error {
	catIDs, err := c.validateCategories()
	if err != nil {
		return err
	}
	if err := c.validateProfiles(catIDs); err != nil {
		return err
	}
	return c.validateTools(catIDs)
}

// validateCategories checks that every category has a unique, non-empty ID and
// returns the set of known IDs for the profile and tool checks.
func (c *Catalog) validateCategories() (map[string]bool, error) {
	catIDs := make(map[string]bool, len(c.Categories))
	for _, cat := range c.Categories {
		if cat.ID == "" {
			return nil, fmt.Errorf("catalog: category with empty id (name %q)", cat.Name)
		}
		if catIDs[cat.ID] {
			return nil, fmt.Errorf("catalog: duplicate category id %q", cat.ID)
		}
		catIDs[cat.ID] = true
	}
	return catIDs, nil
}

// validateProfiles checks that every profile has an ID and only references
// categories that exist.
func (c *Catalog) validateProfiles(catIDs map[string]bool) error {
	for _, p := range c.Profiles {
		if p.ID == "" {
			return fmt.Errorf("catalog: profile with empty id (name %q)", p.Name)
		}
		for _, ref := range p.Categories {
			if !catIDs[ref] {
				return fmt.Errorf("catalog: profile %q references unknown category %q", p.ID, ref)
			}
		}
	}
	return nil
}

// validateTools checks tool IDs for uniqueness and applies the per-tool rules.
func (c *Catalog) validateTools(catIDs map[string]bool) error {
	toolIDs := make(map[string]bool, len(c.Tools))
	for _, t := range c.Tools {
		if t.ID == "" {
			return fmt.Errorf("catalog: tool with empty id (name %q)", t.Name)
		}
		if toolIDs[t.ID] {
			return fmt.Errorf("catalog: duplicate tool id %q", t.ID)
		}
		toolIDs[t.ID] = true

		if err := validateTool(t, catIDs); err != nil {
			return err
		}
	}
	return nil
}

// validateTool checks one tool's category reference, required fields, and
// detect regex.
func validateTool(t model.ToolDef, catIDs map[string]bool) error {
	if !catIDs[t.Category] {
		return fmt.Errorf("catalog: tool %q references unknown category %q", t.ID, t.Category)
	}
	if t.OfficialURL == "" {
		return fmt.Errorf("catalog: tool %q is missing official_url", t.ID)
	}
	if t.Detect.Cmd == "" {
		return fmt.Errorf("catalog: tool %q is missing detect.cmd", t.ID)
	}
	re, err := regexp.Compile(t.Detect.Regex)
	if err != nil {
		return fmt.Errorf("catalog: tool %q detect.regex does not compile: %v", t.ID, err)
	}
	if n := re.NumSubexp(); n != 1 {
		return fmt.Errorf("catalog: tool %q detect.regex must have exactly one capture group, has %d", t.ID, n)
	}
	if t.Repo != "" {
		if parts := strings.Split(t.Repo, "/"); len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("catalog: tool %q repo must be \"org/repo\", got %q", t.ID, t.Repo)
		}
	}
	if t.VersionSource != nil {
		if t.VersionSource.URL == "" {
			return fmt.Errorf("catalog: tool %q has a version_source block with empty url", t.ID)
		}
		vre, err := regexp.Compile(t.VersionSource.Regex)
		if err != nil {
			return fmt.Errorf("catalog: tool %q version_source.regex does not compile: %v", t.ID, err)
		}
		if n := vre.NumSubexp(); n != 1 {
			return fmt.Errorf("catalog: tool %q version_source.regex must have exactly one capture group, has %d", t.ID, n)
		}
	}
	return nil
}
