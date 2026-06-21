// Package digest renders the ranked jobs into the formats the delivery channels
// need: Markdown (Discord / stdout) and HTML (Gmail).
package digest

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/gitraya/jobradar/internal/job"
)

// Markdown renders the digest for Discord and terminal output.
func Markdown(jobs []job.Job, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**JobRadar — %s**\n", now.Format("Mon 02 Jan 2006"))
	if len(jobs) == 0 {
		b.WriteString("_No new matching jobs today._\n")
		return b.String()
	}
	fmt.Fprintf(&b, "_%d new matching role(s)._\n\n", len(jobs))
	for _, j := range jobs {
		fmt.Fprintf(&b, "• **[%s](%s)** — %s\n  %s · %s · %s\n",
			j.Title, j.URL, j.Company, displayLoc(j.Location), posted(j, now), j.Source)
	}
	return b.String()
}

// HTML renders the digest for an email body.
func HTML(jobs []job.Job, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<h2>JobRadar — %s</h2>\n", html.EscapeString(now.Format("Mon 02 Jan 2006")))
	if len(jobs) == 0 {
		b.WriteString("<p><em>No new matching jobs today.</em></p>\n")
		return b.String()
	}
	fmt.Fprintf(&b, "<p><em>%d new matching role(s).</em></p>\n<ul>\n", len(jobs))
	for _, j := range jobs {
		fmt.Fprintf(&b, "  <li><a href=\"%s\"><strong>%s</strong></a> — %s<br>%s · %s · %s</li>\n",
			html.EscapeString(j.URL), html.EscapeString(j.Title), html.EscapeString(j.Company),
			html.EscapeString(displayLoc(j.Location)), posted(j, now), html.EscapeString(j.Source))
	}
	b.WriteString("</ul>\n")
	return b.String()
}

// displayLoc renders a job's location compactly. An empty location is worldwide;
// a long multi-country restriction (common on Himalayas) is summarized as the
// first few regions plus a count so the digest line stays readable.
func displayLoc(loc string) string {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return "Worldwide"
	}
	parts := strings.Split(loc, ", ")
	const maxParts = 3
	if len(parts) > maxParts {
		return fmt.Sprintf("%s +%d more", strings.Join(parts[:maxParts], ", "), len(parts)-maxParts)
	}
	return loc
}

// posted renders a job's age in a compact human form.
func posted(j job.Job, now time.Time) string {
	if j.PostedAt.IsZero() {
		return "recently"
	}
	d := now.Sub(j.PostedAt)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
