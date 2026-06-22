// Package dedupe removes duplicate postings within a single run (F5) and across
// days via a small JSON cache (F6).
package dedupe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// InRun drops repeats within one run, keying on the canonical URL (F5). The
// first occurrence of each URL wins; order is otherwise preserved.
func InRun(jobs []job.Job) []job.Job {
	seen := make(map[string]struct{}, len(jobs))
	out := make([]job.Job, 0, len(jobs))
	for _, j := range jobs {
		key := j.CanonicalURL()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, j)
	}
	return out
}

// Seen is the cross-day cache: canonical URL -> first-seen timestamp (RFC3339).
type Seen struct {
	path    string
	entries map[string]string
}

// LoadSeen reads the cache from path. A missing file yields an empty cache.
func LoadSeen(path string) (*Seen, error) {
	s := &Seen{path: path, entries: map[string]string{}}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &s.entries); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Filter removes jobs already present in the cache, returning the unseen ones.
// It does not record them — call Add after delivery succeeds.
func (s *Seen) Filter(jobs []job.Job) []job.Job {
	out := make([]job.Job, 0, len(jobs))
	for _, j := range jobs {
		if _, ok := s.entries[j.CanonicalURL()]; !ok {
			out = append(out, j)
		}
	}
	return out
}

// Add records jobs as seen, stamped with now.
func (s *Seen) Add(jobs []job.Job, now time.Time) {
	stamp := now.UTC().Format(time.RFC3339)
	for _, j := range jobs {
		s.entries[j.CanonicalURL()] = stamp
	}
}

// Prune drops entries older than maxAge to keep the cache from growing forever.
func (s *Seen) Prune(maxAge time.Duration, now time.Time) {
	cutoff := now.Add(-maxAge)
	for url, stamp := range s.entries {
		if t, err := time.Parse(time.RFC3339, stamp); err == nil && t.Before(cutoff) {
			delete(s.entries, url)
		}
	}
}

// Save writes the cache back to disk as indented JSON, creating the parent
// directory if needed (e.g. a freshly-mounted /data volume).
func (s *Seen) Save() error {
	raw, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(s.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(s.path, append(raw, '\n'), 0o644)
}
