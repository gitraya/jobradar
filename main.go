// Command jobradar runs the fetch → filter → dedupe → rank → deliver pipeline
// once. It is meant to be invoked on a schedule (GitHub Actions cron).
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gitraya/jobradar/internal/config"
	"github.com/gitraya/jobradar/internal/dedupe"
	"github.com/gitraya/jobradar/internal/deliver"
	"github.com/gitraya/jobradar/internal/digest"
	"github.com/gitraya/jobradar/internal/filter"
	"github.com/gitraya/jobradar/internal/job"
	"github.com/gitraya/jobradar/internal/rank"
	"github.com/gitraya/jobradar/internal/source"
)

// seenMaxAge bounds how long a delivered URL stays in the cross-day cache.
const seenMaxAge = 30 * 24 * time.Hour

func main() {
	configPath := flag.String("config", "config.yaml", "path to the config file")
	dryRun := flag.Bool("dry-run", false, "print the digest to stdout; do not deliver or persist")
	seenPath := flag.String("seen", "", "override the config's seen_path (cross-day cache location)")
	flag.Parse()

	log.SetFlags(0)
	if err := run(*configPath, *dryRun, *seenPath); err != nil {
		log.Fatalf("jobradar: %v", err)
	}
}

func run(configPath string, dryRun bool, seenPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	// A -seen override lets a container point the cache at a persistent volume
	// (e.g. /data/seen.json) without editing the committed config.
	if seenPath != "" {
		cfg.SeenPath = seenPath
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 30 * time.Second}
	now := time.Now().UTC()

	// Sources that paginate (Himalayas) stop at the freshness cutoff so they
	// don't walk the whole back catalogue. Zero means "no cutoff".
	var since time.Time
	if cfg.FreshnessHours > 0 {
		since = now.Add(-time.Duration(cfg.FreshnessHours) * time.Hour)
	}

	// 1. Fetch + normalize from the enabled sources concurrently.
	jobs := source.FetchAll(ctx, client, enabledSources(cfg, since))
	log.Printf("fetched %d jobs total", len(jobs))

	// 2. Filters: freshness → role/keyword → location-lock → location allow-list.
	jobs = filter.Fresh(jobs, cfg.FreshnessHours, now)
	jobs = filter.Roles(jobs, cfg.Roles)
	jobs = filter.BlockLocations(jobs, cfg.BlockTerms)
	jobs = filter.AllowLocations(jobs, cfg.Locations)
	log.Printf("%d jobs after filters", len(jobs))

	// 3. Dedupe within the run, then optionally across days.
	jobs = dedupe.InRun(jobs)

	var seen *dedupe.Seen
	if cfg.DedupeAcrossDays {
		seen, err = dedupe.LoadSeen(cfg.SeenPath)
		if err != nil {
			return fmt.Errorf("load seen cache: %w", err)
		}
		jobs = seen.Filter(jobs)
		log.Printf("%d jobs after cross-day dedupe", len(jobs))
	}

	// 4. Rank.
	jobs = rank.Sort(jobs, cfg.RankWorldwideFirst)

	// 5. Render + deliver.
	md := digest.Markdown(jobs, now)
	if dryRun {
		fmt.Println(md)
		return nil
	}
	if err := delivery(ctx, client, cfg, jobs, md, now); err != nil {
		return err
	}

	// 6. Persist the cross-day cache only after successful delivery.
	if seen != nil {
		seen.Add(jobs, now)
		seen.Prune(seenMaxAge, now)
		if err := seen.Save(); err != nil {
			return fmt.Errorf("save seen cache: %w", err)
		}
	}
	return nil
}

// enabledSources returns the sources switched on in config. When the config has
// no `sources:` block at all, every source is enabled.
func enabledSources(cfg *config.Config, since time.Time) []source.Source {
	all := source.All(source.Options{
		RemotiveCategory: cfg.RemotiveCategory,
		RemotiveLocation: cfg.RemotiveLocation,
		Since:            since,
		HimalayasMaxJobs: cfg.HimalayasMaxJobs,
	})
	if len(cfg.Sources) == 0 {
		return all
	}
	out := make([]source.Source, 0, len(all))
	for _, s := range all {
		if cfg.Sources[s.Name()] {
			out = append(out, s)
		}
	}
	return out
}

// delivery sends the digest to every enabled channel, returning the first error.
func delivery(ctx context.Context, client *http.Client, cfg *config.Config, jobs []job.Job, md string, now time.Time) error {
	if cfg.Delivery.Discord.Enabled {
		if err := deliver.Discord(ctx, client, md); err != nil {
			return fmt.Errorf("discord delivery: %w", err)
		}
		log.Printf("delivered to discord")
	}
	if cfg.Delivery.Gmail.Enabled {
		subject := fmt.Sprintf("JobRadar — %d new role(s) — %s", len(jobs), now.Format("02 Jan"))
		if err := deliver.Gmail(digest.HTML(jobs, now), subject); err != nil {
			return fmt.Errorf("gmail delivery: %w", err)
		}
		log.Printf("delivered to gmail")
	}
	if !cfg.Delivery.Discord.Enabled && !cfg.Delivery.Gmail.Enabled {
		log.Printf("no delivery channel enabled; digest:\n%s", md)
	}
	return nil
}
