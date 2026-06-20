package source

import (
	"context"
	"net/http"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// jobicy fetches https://jobicy.com/api/v2/remote-jobs.
type jobicy struct{}

func (jobicy) Name() string { return "jobicy" }

func (s jobicy) Fetch(ctx context.Context, client *http.Client) ([]job.Job, error) {
	var resp struct {
		Jobs []struct {
			JobTitle    string   `json:"jobTitle"`
			CompanyName string   `json:"companyName"`
			URL         string   `json:"url"`
			JobGeo      string   `json:"jobGeo"`
			JobIndustry []string `json:"jobIndustry"`
			PubDate     string   `json:"pubDate"`
		} `json:"jobs"`
	}
	if err := getJSON(ctx, client, "https://jobicy.com/api/v2/remote-jobs", &resp); err != nil {
		return nil, err
	}
	jobs := make([]job.Job, 0, len(resp.Jobs))
	for _, r := range resp.Jobs {
		if r.URL == "" {
			continue
		}
		// Jobicy emits "2024-01-02 15:04:05".
		posted, _ := time.Parse("2006-01-02 15:04:05", r.PubDate)
		jobs = append(jobs, job.Job{
			Source:   s.Name(),
			Title:    r.JobTitle,
			Company:  r.CompanyName,
			URL:      r.URL,
			Location: r.JobGeo,
			Tags:     r.JobIndustry,
			PostedAt: posted.UTC(),
		})
	}
	return jobs, nil
}
