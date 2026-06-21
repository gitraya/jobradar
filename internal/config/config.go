// Package config loads JobRadar's single YAML config file using only the
// standard library. It parses the small YAML subset the PRD's example config
// uses (scalars, block sequences of scalars, and nested mappings) rather than
// pulling in a third-party YAML dependency.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is the fully-resolved configuration after defaults are applied.
type Config struct {
	Roles              []string
	BlockTerms         []string
	Locations          []string        // F4 allow-list: keep only these locations (empty = keep all)
	Sources            map[string]bool // board name -> enabled
	RemotiveCategory   string          // Remotive category slug to query
	RemotiveLocation   string          // Remotive location param to query
	HimalayasMaxJobs   int             // safety cap on Himalayas pagination
	FreshnessHours     int             // 0 disables the freshness filter
	DedupeAcrossDays   bool
	RankWorldwideFirst bool

	Delivery struct {
		Discord struct{ Enabled bool }
		Gmail   struct{ Enabled bool }
	}
	AI struct{ Enabled bool }

	SeenPath string // where the cross-day cache lives; defaults to seen.json
}

// Load reads and parses the config file at path, applying defaults.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	tree, err := parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	root, ok := tree.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("parse config %q: top level must be a mapping", path)
	}

	c := &Config{
		Sources:            map[string]bool{},
		FreshnessHours:     24,
		DedupeAcrossDays:   true,
		RankWorldwideFirst: true,
		SeenPath:           "seen.json",
		RemotiveCategory:   "software-development",
		RemotiveLocation:   "worldwide",
	}
	c.Roles = strSlice(root["roles"])
	c.BlockTerms = strSlice(root["block_terms"])
	c.Locations = strSlice(root["locations"])
	if v, ok := root["remotive_category"]; ok {
		if s, ok := v.(string); ok && s != "" {
			c.RemotiveCategory = s
		}
	}
	if v, ok := root["remotive_location"]; ok {
		if s, ok := v.(string); ok {
			c.RemotiveLocation = s
		}
	}
	if v, ok := root["himalayas_max_jobs"]; ok {
		c.HimalayasMaxJobs = asInt(v, c.HimalayasMaxJobs)
	}
	for name, v := range asMap(root["sources"]) {
		c.Sources[name] = asBool(v)
	}
	if v, ok := root["freshness_hours"]; ok {
		c.FreshnessHours = asInt(v, c.FreshnessHours)
	}
	if v, ok := root["dedupe_across_days"]; ok {
		c.DedupeAcrossDays = asBool(v)
	}
	if v, ok := root["rank_worldwide_first"]; ok {
		c.RankWorldwideFirst = asBool(v)
	}
	if v, ok := root["seen_path"]; ok {
		if s, ok := v.(string); ok && s != "" {
			c.SeenPath = s
		}
	}
	delivery := asMap(root["delivery"])
	c.Delivery.Discord.Enabled = asBool(asMap(delivery["discord"])["enabled"])
	c.Delivery.Gmail.Enabled = asBool(asMap(delivery["gmail"])["enabled"])
	c.AI.Enabled = asBool(asMap(root["ai"])["enabled"])

	return c, nil
}

// --- typed accessors over the generic parse tree ---

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func asBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true") || t == "1" || strings.EqualFold(t, "yes")
	}
	return false
}

func asInt(v any, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return n
		}
	}
	return def
}

func strSlice(v any) []string {
	seq, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(seq))
	for _, item := range seq {
		if s, ok := item.(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}
