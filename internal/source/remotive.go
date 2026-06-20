package source

import (
	"context"
	"net/http"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// remotive fetches https://remotive.com/api/remote-jobs.
type remotive struct{}

func (remotive) Name() string { return "remotive" }

func (s remotive) Fetch(ctx context.Context, client *http.Client) ([]job.Job, error) {
	var resp struct {
		Jobs []struct {
			Title                     string   `json:"title"`
			CompanyName               string   `json:"company_name"`
			URL                       string   `json:"url"`
			CandidateRequiredLocation string   `json:"candidate_required_location"`
			PublicationDate           string   `json:"publication_date"`
			Tags                      []string `json:"tags"`
		} `json:"jobs"`
	}
	if err := getJSON(ctx, client, "https://remotive.com/api/remote-jobs", &resp); err != nil {
		return nil, err
	}
	jobs := make([]job.Job, 0, len(resp.Jobs))
	for _, r := range resp.Jobs {
		if r.URL == "" {
			continue
		}
		// Remotive emits timestamps without a zone, e.g. "2024-01-02T15:04:05".
		posted, _ := time.Parse("2006-01-02T15:04:05", r.PublicationDate)
		jobs = append(jobs, job.Job{
			Source:   s.Name(),
			Title:    r.Title,
			Company:  r.CompanyName,
			URL:      r.URL,
			Location: r.CandidateRequiredLocation,
			Tags:     r.Tags,
			PostedAt: posted.UTC(),
		})
	}
	return jobs, nil
}
