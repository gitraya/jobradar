# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Status

Greenfield. The only content so far is `docs/PRD.md` — the full product spec for **JobRadar**. There is no Go code yet. When implementing, treat the PRD as the source of truth for scope, priorities (Must/Should/Could/Later), and the roadmap (v0.1 → v2.0).

## What JobRadar is

A free, self-hosted CLI that fetches fresh remote software jobs from several free no-auth job-board APIs, filters out roles you can't apply to (stale, off-keyword, location-locked), dedupes them, and delivers a daily digest to Discord and/or Gmail. Designed to run hands-off on GitHub Actions cron at $0/month.

## Hard constraints (from PRD — do not violate)

- **Go standard library only.** Single static binary, no third-party dependencies to download. This rules out common picks like a YAML lib, an HTTP router, or an SMTP package — implement against `net/http`, `net/smtp`, `encoding/json`, etc. If config is YAML per the PRD example, either parse a minimal subset by hand or reconsider as JSON; do not add a module dependency without flagging it.
- **Free by default.** No paid infra or services. Anything needing paid resources ships off by default.
- **AI is off by default and gated behind a toggle** (`ai.enabled`). Features F12–F13 are specified for the future; the product must be fully useful with zero AI. Leave architectural room but don't require it.
- **Config-driven.** Roles, keywords, block-terms, per-source toggles, and delivery channels all come from one config file (see PRD §9) — no code edits for normal use. Every filter, source, and delivery channel is independently toggleable.
- **Secrets** (Discord webhook URL, Gmail app password) come from GitHub Actions Secrets / env — never committed. `seen.json` is the only persisted state.

## Pipeline architecture (PRD §8)

The core is a linear pipeline; build each stage so it can be toggled and tested in isolation:

1. **Sources** — concurrent fetch from job-board APIs (each enable/disable-able). Currently only **Remotive** is wired (queried by `category` + `location`); RemoteOK, Arbeitnow and Jobicy were removed because their feeds were too generic/noisy. The `source.Source` interface + `source.Options` seam keeps re-adding boards cheap. Each source has its own response shape.
2. **Normalize** — map every source's payload into one common `Job` shape. This is the seam that lets new boards be added without touching downstream filters.
3. **Filters** (each toggleable, applied in order): 24h freshness → role/keyword match (keep if ANY) → location-lock block (drop if ANY block-term) → location allow-list (keep only `locations:`, empty location always kept). Remotive's `candidate_required_location` feeds the allow-list since its API doesn't filter location server-side.
4. **Dedupe** — within a run by canonical URL, and across days via `seen.json`.
5. **Rank** — worldwide/anywhere roles first, then newest.
6. **Format** — render the digest as Markdown / HTML / text depending on delivery target.
7. **Deliver** — Discord webhook (HTTP POST) and/or Gmail SMTP (`smtp.gmail.com:587`, app password). Build Discord first per the roadmap.
8. **Persist** — write back `seen.json`.

## Commands

No build config exists yet. Once a `go.mod` and entrypoint are added, the standard flow applies:

```bash
go mod init <module>      # first-time setup
go build ./...            # build
go run .                  # run locally (reads the config file)
go test ./...             # all tests
go test ./internal/filter -run TestFreshness   # single package / single test
go vet ./...              # vet
gofmt -l .                # list unformatted files (gofmt -w . to fix)
```

When wiring CI, the daily run is a GitHub Actions cron invoking the built binary with secrets injected as env vars.
