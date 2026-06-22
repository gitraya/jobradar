# JobRadar

A free, self-hosted assistant that fetches fresh remote software jobs,
filters out everything you can't actually apply to, dedupes, and delivers a
clean daily digest to **Discord** and/or **Gmail** — no AI required. Built in
Go with the standard library only (single static binary, zero dependencies).
See [`docs/PRD.md`](docs/PRD.md) for the full spec.

## How it works

```
sources → normalize → filters (24h · roles · block-terms · location allow-list)
   → dedupe (in-run + seen.json) → rank (worldwide first) → digest → deliver
```

Sources (free, no-auth APIs):

- **[Remotive](https://remotive.com)** — queried by `software-development`
  category. Low-volume; a single call.
- **[Himalayas](https://himalayas.app)** — `jobs/api`, newest-first and
  high-volume, so it pages by offset and stops at the freshness cutoff
  (`himalayas_max_jobs` bounds how far back one run pages). Empty
  `locationRestrictions` means worldwide.

Neither API filters location server-side, so the **location allow-list** keeps
only roles open to your regions (`locations:` — worldwide, asia, apac,
indonesia). The `internal/source` seam makes adding more boards cheap.

> **Note:** if a run shows few/no jobs, widen `freshness_hours` (e.g. `168` for
> a week) — Remotive especially is often quiet within 24h.

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

## Run on a server (Docker)

```bash
cp .env.example .env          # fill in DISCORD_WEBHOOK_URL / GMAIL_* secrets
cp config.example.yaml config.yaml   # edit roles / locations / sources
docker compose up -d --build  # daemon: runs daily at RUN_AT (UTC) in compose
docker compose logs -f        # watch runs
```

The container has two modes (see `docker-compose.yml` / `docker-entrypoint.sh`):

- **Daemon** — set `RUN_AT="01:00"` (UTC); the container stays up and runs once
  a day. This is the compose default. `seen.json` persists in the `jobradar_state`
  named volume across restarts.
- **Run once** — unset `RUN_AT` and drive it from host cron / a systemd timer:
  ```bash
  docker compose run --rm jobradar           # one delivery
  DRY_RUN=true docker compose run --rm jobradar   # fetch + print, deliver nothing
  ```

`config.yaml` is bind-mounted read-only — edit it on the host, no rebuild needed.
The image is a single static binary on Alpine (stdlib only, no deps). `seen.json`
persists in the `/data` volume (`SEEN_PATH=/data/seen.json`).

### Coolify

JobRadar deploys on [Coolify](https://coolify.io) as a **Docker Compose** resource:

1. **New Resource → Docker Compose**, point it at this Git repo (it uses the
   committed `docker-compose.yml`). The image bakes `config.yaml`, so no host
   bind is required.
2. In **Environment Variables**, set your secrets and schedule:
   `DISCORD_WEBHOOK_URL`, optionally `GMAIL_USERNAME` / `GMAIL_APP_PASSWORD` /
   `GMAIL_TO`, and `RUN_AT` (UTC, e.g. `01:00`). Compose interpolates these in.
3. The `jobradar_state` volume keeps `seen.json` across redeploys — Coolify
   persists named volumes automatically.
4. **Deploy.** The container stays up and runs daily at `RUN_AT` (watch
   **Logs**). To change `roles`/`locations`, edit `config.yaml`, push, redeploy.

The container self-schedules via `RUN_AT`, so you don't need Coolify's own
Scheduled Tasks. Use `RUN_ON_START=true` (or temporarily set `RUN_AT` to a
minute from now) if you want to trigger a run immediately to verify delivery.

## Scheduled runs on GitHub Actions (free)

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
