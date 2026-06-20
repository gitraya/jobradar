package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExampleConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	yaml := `
roles:
  - software engineer
  - golang

block_terms:
  - us only
  - h-1b

sources:
  remoteok: true
  remotive: false

freshness_hours: 12
dedupe_across_days: false
rank_worldwide_first: true

delivery:
  discord:
    enabled: true
  gmail:
    enabled: false

ai:
  enabled: false
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.Roles) != 2 || cfg.Roles[0] != "software engineer" || cfg.Roles[1] != "golang" {
		t.Errorf("roles = %v", cfg.Roles)
	}
	if len(cfg.BlockTerms) != 2 || cfg.BlockTerms[1] != "h-1b" {
		t.Errorf("block_terms = %v", cfg.BlockTerms)
	}
	if !cfg.Sources["remoteok"] || cfg.Sources["remotive"] {
		t.Errorf("sources = %v", cfg.Sources)
	}
	if cfg.FreshnessHours != 12 {
		t.Errorf("freshness_hours = %d, want 12", cfg.FreshnessHours)
	}
	if cfg.DedupeAcrossDays {
		t.Errorf("dedupe_across_days = true, want false")
	}
	if !cfg.RankWorldwideFirst {
		t.Errorf("rank_worldwide_first = false, want true")
	}
	if !cfg.Delivery.Discord.Enabled || cfg.Delivery.Gmail.Enabled || cfg.AI.Enabled {
		t.Errorf("delivery/ai toggles wrong: %+v / %+v", cfg.Delivery, cfg.AI)
	}
	if cfg.SeenPath != "seen.json" {
		t.Errorf("seen_path default = %q", cfg.SeenPath)
	}
}

func TestDefaultsWhenOmitted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("roles:\n  - go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.FreshnessHours != 24 || !cfg.DedupeAcrossDays || !cfg.RankWorldwideFirst {
		t.Errorf("defaults not applied: %+v", cfg)
	}
}
