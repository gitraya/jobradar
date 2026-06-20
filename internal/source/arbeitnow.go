package source

import (
	"context"
	"net/http"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// arbeitnow fetches https://www.arbeitnow.com/api/job-board-api.
type arbeitnow struct{}

func (arbeitnow) Name() string { return "arbeitnow" }

func (s arbeitnow) Fetch(ctx context.Context, client *http.Client) ([]job.Job, error) {
	var resp struct {
		Data []struct {
			Title       string   `json:"title"`
			CompanyName string   `json:"company_name"`
			URL         string   `json:"url"`
			Location    string   `json:"location"`
			Remote      bool     `json:"remote"`
			Tags        []string `json:"tags"`
			CreatedAt   int64    `json:"created_at"`
		} `json:"data"`
	}
	if err := getJSON(ctx, client, "https://www.arbeitnow.com/api/job-board-api", &resp); err != nil {
		return nil, err
	}
	jobs := make([]job.Job, 0, len(resp.Data))
	for _, r := range resp.Data {
		if r.URL == "" {
			continue
		}
		var posted time.Time
		if r.CreatedAt > 0 {
			posted = time.Unix(r.CreatedAt, 0).UTC()
		}
		jobs = append(jobs, job.Job{
			Source:   s.Name(),
			Title:    r.Title,
			Company:  r.CompanyName,
			URL:      r.URL,
			Location: r.Location,
			Tags:     r.Tags,
			PostedAt: posted,
		})
	}
	return jobs, nil
}
