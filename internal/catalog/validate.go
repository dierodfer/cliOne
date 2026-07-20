package catalog

import (
	"fmt"
	"regexp"
)

// validate enforces the catalog schema rules, naming the offending tool ID in
// every error.
func (c *Catalog) validate() error {
	catIDs := make(map[string]bool, len(c.Categories))
	for _, cat := range c.Categories {
		if cat.ID == "" {
			return fmt.Errorf("catalog: category with empty id (name %q)", cat.Name)
		}
		if catIDs[cat.ID] {
			return fmt.Errorf("catalog: duplicate category id %q", cat.ID)
		}
		catIDs[cat.ID] = true
	}

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

	toolIDs := make(map[string]bool, len(c.Tools))
	for _, t := range c.Tools {
		if t.ID == "" {
			return fmt.Errorf("catalog: tool with empty id (name %q)", t.Name)
		}
		if toolIDs[t.ID] {
			return fmt.Errorf("catalog: duplicate tool id %q", t.ID)
		}
		toolIDs[t.ID] = true

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
		if t.Update != nil && t.Update.Cmd == "" {
			return fmt.Errorf("catalog: tool %q has an update block with empty cmd", t.ID)
		}
	}
	return nil
}
