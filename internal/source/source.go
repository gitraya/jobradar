// Package source fetches postings from free, no-auth job-board APIs and
// normalizes each board's response into the common job.Job shape.
package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// userAgent is sent on every request. RemoteOK (and others) reject the default
// Go HTTP user agent, so a non-empty identifier is required.
const userAgent = "JobRadar/0.1 (+https://github.com/gitraya/jobradar)"

// Source is one job board. Fetch returns already-normalized jobs.
type Source interface {
	Name() string
	Fetch(ctx context.Context, client *http.Client) ([]job.Job, error)
}

// Options carries source-specific fetch settings derived from config.
type Options struct {
	RemotiveCategory string    // Remotive category slug, e.g. software-development
	RemotiveLocation string    // Remotive location param, e.g. worldwide
	Since            time.Time // freshness cutoff for paginated sources (zero = none)
	HimalayasMaxJobs int       // safety cap on Himalayas pagination (0 = default)
}

// All returns every known source. main enables/disables them via config.
func All(opts Options) []Source {
	return []Source{
		remotive{category: opts.RemotiveCategory, location: opts.RemotiveLocation},
		himalayas{since: opts.Since, maxJobs: opts.HimalayasMaxJobs},
	}
}

// FetchAll fetches the given sources concurrently. A failing source is logged
// and skipped rather than aborting the whole run, so one flaky board never
// blocks the digest. Results are sorted by source name for deterministic output.
func FetchAll(ctx context.Context, client *http.Client, sources []Source) []job.Job {
	var (
		mu  sync.Mutex
		all []job.Job
		wg  sync.WaitGroup
	)
	for _, s := range sources {
		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			jobs, err := s.Fetch(ctx, client)
			if err != nil {
				log.Printf("source %s: %v", s.Name(), err)
				return
			}
			mu.Lock()
			all = append(all, jobs...)
			mu.Unlock()
			log.Printf("source %s: %d jobs", s.Name(), len(jobs))
		}(s)
	}
	wg.Wait()
	sort.SliceStable(all, func(i, j int) bool { return all[i].Source < all[j].Source })
	return all
}

// getJSON performs a GET with the required headers and decodes the body into v.
func getJSON(ctx context.Context, client *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
