// Package rank orders the digest: worldwide/anywhere roles first (F10), then
// newest postings.
package rank

import (
	"sort"
	"strings"

	"github.com/gitraya/jobradar/internal/job"
)

// worldwideMarkers identify location-unrestricted roles worth surfacing first.
var worldwideMarkers = []string{"worldwide", "anywhere", "global"}

// Sort orders jobs in place and returns them. When worldwideFirst is set,
// worldwide/anywhere roles are grouped ahead of the rest; within each group
// (and when the toggle is off) jobs are ordered newest-first.
func Sort(jobs []job.Job, worldwideFirst bool) []job.Job {
	sort.SliceStable(jobs, func(i, k int) bool {
		if worldwideFirst {
			wi, wk := isWorldwide(jobs[i]), isWorldwide(jobs[k])
			if wi != wk {
				return wi // worldwide sorts before non-worldwide
			}
		}
		return jobs[i].PostedAt.After(jobs[k].PostedAt)
	})
	return jobs
}

func isWorldwide(j job.Job) bool {
	loc := strings.ToLower(strings.TrimSpace(j.Location))
	if loc == "" {
		return true // no restriction stated
	}
	for _, m := range worldwideMarkers {
		if strings.Contains(loc, m) {
			return true
		}
	}
	return false
}
