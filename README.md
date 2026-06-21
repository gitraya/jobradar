# JobRadar

A free, self-hosted assistant that fetches fresh remote software jobs,
filters out everything you can't actually apply to, dedupes, and delivers a
clean daily digest to **Discord** and/or **Gmail** — no AI required. Built in
Go with the standard library only (single static binary, zero dependencies).
See [`docs/PRD.md`](docs/PRD.md) for the full spec.

## How it works

```
Remotive → normalize → filters (24h · roles · block-terms · location allow-list)
   → dedupe (in-run + seen.json) → rank (worldwide first) → digest → deliver
```

Source: [Remotive](https://remotive.com) — a free, no-auth API. We query its
`software-development` category and keep only worldwide / APAC-friendly roles
(`candidate_required_location` in worldwide, asia, apac, indonesia). The source
seam (`internal/source`) is built so other boards can be added back later.

> **Note:** Remotive's filtered feed is low-volume — often nothing is newer than
> 24h. If a run shows no jobs, widen `freshness_hours` (e.g. `168` for a week).

## Quick start

```bash
cp config.example.yaml config.yaml   # then edit roles / block_terms / toggles
go run . -config config.yaml -dry-run # fetch + filter, print digest, deliver nothing
```

Drop `-dry-run` to actually deliver. Delivery secrets come from the environment,
never the config file:

| Channel | Env vars |
|---------|----------|
| Discord | `DISCORD_WEBHOOK_URL` |
| Gmail   | `GMAIL_USERNAME`, `GMAIL_APP_PASSWORD` (a Gmail App Password — needs 2FA), `GMAIL_TO` (optional, defaults to sender) |

```bash
DISCORD_WEBHOOK_URL=... go run . -config config.yaml
```

## Configuration

All behaviour is config-driven (`config.yaml`) — no code edits for normal use.
Every source, filter, and delivery channel is independently toggleable. See
[`config.example.yaml`](config.example.yaml) for the annotated, complete set.

## Scheduled runs (free)

[`.github/workflows/digest.yml`](.github/workflows/digest.yml) runs the binary
daily on GitHub Actions cron and commits the updated `seen.json` back to the
repo (cross-day dedupe). Add `DISCORD_WEBHOOK_URL` (and optionally the Gmail
vars) as **Actions repository secrets**.

## Development

```bash
go build ./...        # build
go test ./...         # all tests
go vet ./... && gofmt -l .   # lint / format check
```

## Status

v0.1–v0.5 of the [roadmap](docs/PRD.md#11-roadmap) — fetch, filters, dedupe,
ranking, Discord + Gmail delivery, and the daily schedule. The AI tier
(F12–F13) is specified but ships off by default (`ai.enabled: false`).
