package source

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// himalayasPageSize is the fixed page size of the Himalayas API — it caps each
// response at 20 regardless of the requested limit, so we paginate by offset.
const himalayasPageSize = 20

// himalayasMaxJobsDefault bounds pagination when no freshness cutoff is given,
// so a run can never walk the entire (80k+) back catalogue.
const himalayasMaxJobsDefault = 2000

// himalayas fetches https://himalayas.app/jobs/api. The feed is newest-first and
// high-volume, with no server-side keyword/location filtering, so we page back
// by offset and stop once postings predate `since` (the freshness cutoff). An
// empty `since` falls back to the maxJobs cap. A job with no locationRestrictions
// is open worldwide; that empty location flows to the downstream allow-list.
type himalayas struct {
	since   time.Time // stop paging once jobs are older than this (zero = no cutoff)
	maxJobs int       // hard safety cap on how many to pull
}

func (himalayas) Name() string { return "himalayas" }

type himalayasJob struct {
	Title                string   `json:"title"`
	CompanyName          string   `json:"companyName"`
	ApplicationLink      string   `json:"applicationLink"`
	Guid                 string   `json:"guid"`
	LocationRestrictions []string `json:"locationRestrictions"`
	Categories           []string `json:"categories"`
	PubDate              int64    `json:"pubDate"`
}

func (s himalayas) Fetch(ctx context.Context, client *http.Client) ([]job.Job, error) {
	maxJobs := s.maxJobs
	if maxJobs <= 0 {
		maxJobs = himalayasMaxJobsDefault
	}

	var jobs []job.Job
	for offset := 0; offset < maxJobs; offset += himalayasPageSize {
		page, err := s.fetchPage(ctx, client, offset)
		if err != nil {
			if len(jobs) > 0 {
				break // keep what we already gathered; a later page failed
			}
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		reachedCutoff := false
		for _, r := range page {
			url := r.ApplicationLink
			if url == "" {
				url = r.Guid
			}
			if r.Title == "" || url == "" {
				continue
			}
			var posted time.Time
			if r.PubDate > 0 {
				posted = time.Unix(r.PubDate, 0).UTC()
			}
			if !s.since.IsZero() && !posted.IsZero() && posted.Before(s.since) {
				reachedCutoff = true
				break
			}
			jobs = append(jobs, job.Job{
				Source:   s.Name(),
				Title:    r.Title,
				Company:  r.CompanyName,
				URL:      url,
				Location: strings.Join(r.LocationRestrictions, ", "),
				Tags:     r.Categories,
				PostedAt: posted,
			})
		}
		if reachedCutoff || len(page) < himalayasPageSize {
			break
		}
	}
	return jobs, nil
}

func (s himalayas) fetchPage(ctx context.Context, client *http.Client, offset int) ([]himalayasJob, error) {
	endpoint := fmt.Sprintf("https://himalayas.app/jobs/api?limit=%d&offset=%d", himalayasPageSize, offset)
	var resp struct {
		Jobs []himalayasJob `json:"jobs"`
	}
	if err := getJSON(ctx, client, endpoint, &resp); err != nil {
		return nil, err
	}
	return resp.Jobs, nil
}
