package source

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// remotive fetches https://remotive.com/api/remote-jobs, narrowed to a category
// and location (e.g. category=software-development&location=worldwide). The
// location param is a loose hint — Remotive still returns other regions — so
// candidate_required_location is filtered downstream by the location allow-list.
type remotive struct {
	category string
	location string
}

func (remotive) Name() string { return "remotive" }

func (s remotive) Fetch(ctx context.Context, client *http.Client) ([]job.Job, error) {
	endpoint := "https://remotive.com/api/remote-jobs"
	if q := s.query(); q != "" {
		endpoint += "?" + q
	}

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
	if err := getJSON(ctx, client, endpoint, &resp); err != nil {
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

func (s remotive) query() string {
	v := url.Values{}
	if s.category != "" {
		v.Set("category", s.category)
	}
	if s.location != "" {
		v.Set("location", s.location)
	}
	return v.Encode()
}
