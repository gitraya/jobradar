// Package filter implements the toggleable filter stages: freshness (F2),
// role/keyword match (F3), and the location-lock block (F4).
package filter

import (
	"strings"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// Fresh keeps only jobs posted within the last freshnessHours. A non-positive
// freshnessHours disables the filter. Jobs with no known posting time are
// dropped, since the goal is strictly fresh postings.
func Fresh(jobs []job.Job, freshnessHours int, now time.Time) []job.Job {
	if freshnessHours <= 0 {
		return jobs
	}
	cutoff := now.Add(-time.Duration(freshnessHours) * time.Hour)
	out := jobs[:0:0]
	for _, j := range jobs {
		if !j.PostedAt.IsZero() && j.PostedAt.After(cutoff) {
			out = append(out, j)
		}
	}
	return out
}

// Roles keeps jobs whose text matches ANY of the role keywords (F3). An empty
// roles list disables the filter (everything passes).
func Roles(jobs []job.Job, roles []string) []job.Job {
	terms := lower(roles)
	if len(terms) == 0 {
		return jobs
	}
	out := jobs[:0:0]
	for _, j := range jobs {
		if containsAny(j.Haystack(), terms) {
			out = append(out, j)
		}
	}
	return out
}

// BlockLocations drops jobs whose text matches ANY block term (F4), e.g.
// "us only", "authorized to work in", "h-1b". An empty list disables the filter.
func BlockLocations(jobs []job.Job, blockTerms []string) []job.Job {
	terms := lower(blockTerms)
	if len(terms) == 0 {
		return jobs
	}
	out := jobs[:0:0]
	for _, j := range jobs {
		if !containsAny(j.Haystack(), terms) {
			out = append(out, j)
		}
	}
	return out
}

// AllowLocations keeps only jobs whose location is acceptable: an empty
// location (unspecified ⇒ treated as worldwide) or one matching ANY allowed
// term (e.g. "worldwide", "anywhere", "indonesia", "asia"). This mirrors
// RemoteOK's site-side location filter, which the API does not apply.
// An empty allow-list disables the filter. Unlike BlockLocations it matches
// only against the location field, not tags/title.
func AllowLocations(jobs []job.Job, allowed []string) []job.Job {
	terms := lower(allowed)
	if len(terms) == 0 {
		return jobs
	}
	out := jobs[:0:0]
	for _, j := range jobs {
		loc := strings.ToLower(strings.TrimSpace(j.Location))
		if loc == "" || containsAny(loc, terms) {
			out = append(out, j)
		}
	}
	return out
}

func containsAny(haystack string, terms []string) bool {
	for _, t := range terms {
		if t != "" && strings.Contains(haystack, t) {
			return true
		}
	}
	return false
}

func lower(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			out = append(out, s)
		}
	}
	return out
}
