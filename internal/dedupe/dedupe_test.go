package dedupe

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

func TestSeenSaveCreatesParentDirAndRoundTrips(t *testing.T) {
	// Mirrors a fresh /data volume: the parent dir doesn't exist yet.
	path := filepath.Join(t.TempDir(), "data", "seen.json")

	s, err := LoadSeen(path)
	if err != nil {
		t.Fatalf("LoadSeen (missing file): %v", err)
	}
	jobs := []job.Job{{URL: "https://example.com/jobs/1?utm=x"}}
	if got := s.Filter(jobs); len(got) != 1 {
		t.Fatalf("first Filter dropped a never-seen job: %d", len(got))
	}
	s.Add(jobs, time.Now())
	if err := s.Save(); err != nil {
		t.Fatalf("Save into a non-existent dir should mkdir: %v", err)
	}

	// Reload and confirm the job is now remembered (and URL-canonicalized).
	s2, err := LoadSeen(path)
	if err != nil {
		t.Fatalf("LoadSeen (existing file): %v", err)
	}
	if got := s2.Filter(jobs); len(got) != 0 {
		t.Fatalf("reloaded cache should treat the job as seen, kept %d", len(got))
	}
}
