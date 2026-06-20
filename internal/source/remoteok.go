package source

import (
	"context"
	"net/http"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// remoteOK fetches https://remoteok.com/api. The response is a JSON array whose
// first element is a legal/metadata notice (no position) which we skip.
type remoteOK struct{}

func (remoteOK) Name() string { return "remoteok" }

func (s remoteOK) Fetch(ctx context.Context, client *http.Client) ([]job.Job, error) {
	var raw []struct {
		Position string   `json:"position"`
		Company  string   `json:"company"`
		Tags     []string `json:"tags"`
		Location string   `json:"location"`
		URL      string   `json:"url"`
		Epoch    int64    `json:"epoch"`
	}
	if err := getJSON(ctx, client, "https://remoteok.com/api", &raw); err != nil {
		return nil, err
	}
	jobs := make([]job.Job, 0, len(raw))
	for _, r := range raw {
		if r.Position == "" || r.URL == "" {
			continue // metadata notice or malformed entry
		}
		var posted time.Time
		if r.Epoch > 0 {
			posted = time.Unix(r.Epoch, 0).UTC()
		}
		jobs = append(jobs, job.Job{
			Source:   s.Name(),
			Title:    r.Position,
			Company:  r.Company,
			URL:      r.URL,
			Location: r.Location,
			Tags:     r.Tags,
			PostedAt: posted,
		})
	}
	return jobs, nil
}
