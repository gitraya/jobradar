package filter

import (
	"testing"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

func sample() []job.Job {
	now := time.Now().UTC()
	return []job.Job{
		{Title: "Senior Golang Engineer", Location: "Worldwide", PostedAt: now.Add(-2 * time.Hour)},
		{Title: "React Developer", Location: "US only", PostedAt: now.Add(-3 * time.Hour)},
		{Title: "Designer", Location: "Worldwide", PostedAt: now.Add(-1 * time.Hour)},
		{Title: "Backend Engineer", Location: "Anywhere", PostedAt: now.Add(-72 * time.Hour)},
		{Title: "Old Go Role", Location: "Remote", PostedAt: time.Time{}}, // unknown time
	}
}

func TestFresh(t *testing.T) {
	now := time.Now().UTC()
	got := Fresh(sample(), 24, now)
	// Drops the 72h-old role and the unknown-time role.
	if len(got) != 3 {
		t.Fatalf("Fresh kept %d, want 3: %+v", len(got), got)
	}
	got = Fresh(sample(), 0, now) // disabled
	if len(got) != 5 {
		t.Fatalf("disabled Fresh kept %d, want 5", len(got))
	}
}

func TestRoles(t *testing.T) {
	got := Roles(sample(), []string{"golang", "engineer"})
	for _, j := range got {
		if j.Title == "Designer" || j.Title == "React Developer" {
			t.Errorf("Roles wrongly kept %q", j.Title)
		}
	}
	// Matches: "Senior Golang Engineer" and "Backend Engineer".
	// "Old Go Role" does NOT match — "go" is not "golang".
	if len(got) != 2 {
		t.Fatalf("Roles kept %d, want 2: %+v", len(got), got)
	}
}

func TestBlockLocations(t *testing.T) {
	got := BlockLocations(sample(), []string{"us only"})
	for _, j := range got {
		if j.Title == "React Developer" {
			t.Fatalf("BlockLocations failed to drop US-only role")
		}
	}
	if len(got) != 4 {
		t.Fatalf("BlockLocations kept %d, want 4", len(got))
	}
}
