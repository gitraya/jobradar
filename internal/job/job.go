// Package job defines the common Job shape that every source normalizes into,
// plus helpers shared by the filter/dedupe/rank stages.
package job

import (
	"net/url"
	"strings"
	"time"
)

// Job is the normalized representation of a posting, independent of which
// board it came from. The pipeline only ever works with this shape.
type Job struct {
	Source   string    // board the job came from, e.g. "remoteok"
	Title    string    // role title
	Company  string    // hiring company
	URL      string    // original posting URL
	Location string    // raw location string from the source ("", "Worldwide", "US only", ...)
	Tags     []string  // stack/skill tags when the source provides them
	PostedAt time.Time // publish time; zero when the source gave none
}

// CanonicalURL returns a normalized form of the posting URL used as the dedupe
// key. It lowercases the scheme/host, drops the query and fragment, and trims a
// trailing slash so the same job posted with tracking params collapses to one.
func (j Job) CanonicalURL() string {
	u, err := url.Parse(strings.TrimSpace(j.URL))
	if err != nil || u.Host == "" {
		return strings.TrimSpace(strings.ToLower(j.URL))
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}

// Haystack is the lowercased text that the keyword and location-lock filters
// search against: title, location and tags joined together.
func (j Job) Haystack() string {
	parts := make([]string, 0, len(j.Tags)+2)
	parts = append(parts, j.Title, j.Location)
	parts = append(parts, j.Tags...)
	return strings.ToLower(strings.Join(parts, " "))
}
