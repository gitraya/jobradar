package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// roundTripFunc lets us redirect the source's hard-coded himalayas.app URL to a
// test server by rewriting the request host.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHimalayasPaginationStopsAtCutoff(t *testing.T) {
	now := time.Now().UTC()
	// 3 pages of 20. Page 0 & 1 are fresh; page 2 is all older than the cutoff.
	pageJob := func(idx int, ageHours int) himalayasJob {
		return himalayasJob{
			Title:           fmt.Sprintf("Engineer %d", idx),
			CompanyName:     "Acme",
			ApplicationLink: fmt.Sprintf("https://himalayas.app/jobs/%d", idx),
			PubDate:         now.Add(-time.Duration(ageHours) * time.Hour).Unix(),
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var jobs []himalayasJob
		for i := 0; i < himalayasPageSize; i++ {
			idx := offset + i
			age := 1 // fresh
			if offset >= 2*himalayasPageSize {
				age = 48 // page 2: stale
			}
			jobs = append(jobs, pageJob(idx, age))
		}
		json.NewEncoder(w).Encode(struct {
			Jobs []himalayasJob `json:"jobs"`
		}{jobs})
	}))
	defer srv.Close()

	// Rewrite himalayas.app -> test server.
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = srv.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(r)
	})}

	s := himalayas{since: now.Add(-24 * time.Hour), maxJobs: 10000}
	jobs, err := s.Fetch(context.Background(), client)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	// Should keep the 40 fresh jobs (pages 0,1) and stop at the first stale one.
	if len(jobs) != 2*himalayasPageSize {
		t.Fatalf("got %d jobs, want %d (should stop at the cutoff)", len(jobs), 2*himalayasPageSize)
	}
}

func TestHimalayasRespectsMaxJobs(t *testing.T) {
	now := time.Now().UTC()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var jobs []himalayasJob
		for i := 0; i < himalayasPageSize; i++ {
			jobs = append(jobs, himalayasJob{
				Title:           "Engineer",
				ApplicationLink: fmt.Sprintf("https://himalayas.app/jobs/%s-%d", r.URL.Query().Get("offset"), i),
				PubDate:         now.Unix(), // always fresh, so only maxJobs bounds it
			})
		}
		json.NewEncoder(w).Encode(struct {
			Jobs []himalayasJob `json:"jobs"`
		}{jobs})
	}))
	defer srv.Close()
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = srv.Listener.Addr().String()
		return http.DefaultTransport.RoundTrip(r)
	})}

	s := himalayas{maxJobs: 40} // no cutoff
	jobs, err := s.Fetch(context.Background(), client)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(jobs) != 40 {
		t.Fatalf("got %d jobs, want 40 (maxJobs cap)", len(jobs))
	}
}
