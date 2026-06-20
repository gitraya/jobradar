# JobRadar — Product Requirements Document

**Working title:** JobRadar
**Author:** Raya
**Status:** Draft v1 (hobby project)
**One-liner:** A free, self-hosted assistant that fetches fresh remote software jobs from multiple boards, filters out everything you can't actually apply to, and delivers a clean daily digest to Discord or Gmail — no AI required.

---

## 1. Problem

Searching for remote software roles as an Indonesia-based engineer means doing the same manual job every day:

- Visiting 5+ job boards one by one (LinkedIn, Indeed, JobStreet, We Work Remotely, etc.).
- Mentally filtering out the noise — most "remote" roles are secretly location-locked ("US-only", "must be authorized to work in X", visa-required).
- Reading duplicates posted on multiple boards.
- Doing this for ~1 hour, every day, and still missing fresh postings.

This is a repetitive aggregate-filter-dedupe task — exactly the kind of thing software should do, not a human.

## 2. Goal

Replace that hour with a single pre-filtered digest. The tool fetches jobs posted in the **last 24 hours** from free sources, keeps only roles matching a **user-defined role/keyword list**, drops anything **location-locked**, removes duplicates, and **delivers** the result to a channel the user already checks (Discord or Gmail). Everything runs on free infrastructure.

### Non-goals (keep it simple)
- No web UI in v1 — config file + scheduled run is enough.
- No database in v1 — a small `seen.json` cache is enough to avoid repeats.
- No paid services, ever. If a feature needs paid infra, it stays optional and off by default.

## 3. Principles

1. **Free by default.** Free APIs, free compute (GitHub Actions), free delivery (Discord webhook / Gmail SMTP).
2. **AI is optional and off by default.** The product is fully useful with zero AI. AI features are designed-in but gated behind a toggle, to be enabled later when budget allows.
3. **Config-driven.** The user controls roles, keywords, block-terms, sources, and delivery through one config file — no code edits needed for normal use.
4. **Everything toggleable.** Each filter, source, and delivery channel can be switched on/off independently.

## 4. Users

- **Primary:** A single engineer (you) running it for your own search.
- **Secondary (later):** Other job-seekers who clone the repo and drop in their own config — e.g. your sibling learning frontend.

## 5. Features

| # | Feature | Priority | Toggleable | Needs AI |
|---|---------|----------|------------|----------|
| F1 | Fetch jobs from free no-auth APIs (RemoteOK, Remotive, Arbeitnow, Jobicy) | Must | per-source | No |
| F2 | Keep only jobs posted in the **last 24h** | Must | Yes | No |
| F3 | **Role/keyword filter** — user inputs the roles/stack they want | Must | — (core) | No |
| F4 | **Location-lock filter** — drop US-only / visa-required / region-locked | Must | Yes | No |
| F5 | Dedupe within a run (by canonical URL) | Must | No | No |
| F6 | Dedupe across days (`seen.json` cache, so no repeats tomorrow) | Should | Yes | No |
| F7 | Deliver digest to **Discord** (webhook) | Must | Yes | No |
| F8 | Deliver digest to **Gmail** (SMTP) | Should | Yes | No |
| F9 | Run automatically on a daily schedule (GitHub Actions cron) | Must | — | No |
| F10 | Rank "worldwide / anywhere" roles to the top | Should | Yes | No |
| F11 | **CV keyword extraction (lite)** — pull keywords from your CV text to auto-build the F3 list, no AI | Could | Yes | No |
| F12 | **CV semantic match** — upload CV → convert to MD → rank jobs by fit | Later | Yes | **Yes** |
| F13 | **AI summary / "why it fits"** per job | Later | Yes | **Yes** |

The AI tier (F12–F13) is the part you can't run yet — and that's fine. It's specified now so the architecture leaves room for it, but it ships off by default.

## 6. AI tier (future, gated)

When API budget exists, enabling the AI toggle unlocks:
- Upload CV (PDF/DOCX) → parse to Markdown → embed.
- Score each fetched job against the CV and rank by semantic fit.
- A one-line "why this matches you" per job.

Until then, **F11 (lite CV keyword extraction)** gives ~80% of the value with 0% of the cost: it reads your CV text, pulls out stack/role keywords deterministically, and feeds them into the normal keyword filter.

## 7. Tech stack (all free)

- **Language:** Go (single static binary, standard library only, no dependencies to download).
- **Source/hosting:** GitHub repo.
- **Scheduler/compute:** GitHub Actions (free cron + runners).
- **Delivery:**
  - *Discord* — incoming webhook URL, plain HTTP POST. Easiest, recommended first.
  - *Gmail* — SMTP (`smtp.gmail.com:587`) using a Gmail **App Password** (requires 2FA on the account).
- **State:** `seen.json` committed back to the repo, or stored in the Actions cache.
- **Secrets:** Discord webhook URL and Gmail app password stored as **GitHub Actions Secrets** — never committed to the repo.

## 8. Architecture

```
                ┌──────────── sources (concurrent fetch) ───────────┐
                │  RemoteOK   Remotive   Arbeitnow   Jobicy   (...)  │
                └───────────────────────┬───────────────────────────┘
                                        ▼
                                   Normalize  → common Job shape
                                        ▼
   Filters (each toggleable):  24h fresh → role/keyword → location-lock
                                        ▼
                          Dedupe (in-run + seen.json)
                                        ▼
                          Rank (worldwide first, newest next)
                                        ▼
                        Format digest (Markdown / HTML / text)
                                        ▼
                 Deliver  →  Discord webhook  and/or  Gmail SMTP
                                        ▼
                            Persist seen.json
```

## 9. Example config

```yaml
roles:                       # F3 — what you want (match ANY)
  - software engineer
  - full stack
  - golang
  - react
  - node

block_terms:                 # F4 — auto-drop (match ANY)
  - us only
  - authorized to work in
  - h-1b
  - eu only

sources:                     # F1 — enable/disable per board
  remoteok: true
  remotive: true
  arbeitnow: true
  jobicy: true

freshness_hours: 24          # F2
dedupe_across_days: true     # F6
rank_worldwide_first: true   # F10

delivery:
  discord:
    enabled: true            # F7 — webhook URL from a GitHub Secret
  gmail:
    enabled: false           # F8 — app password from a GitHub Secret

ai:
  enabled: false             # F12/F13 — keep off until budget exists
```

## 10. Delivery: Discord vs Gmail

| | Discord | Gmail |
|---|---------|-------|
| Setup effort | Very low — create a channel webhook, paste URL | Medium — enable 2FA, create an App Password |
| Cost | Free | Free |
| Best for | Quick daily skim on phone/desktop | A searchable archive in your inbox |
| Recommendation | **Build first** | Add second |

## 11. Roadmap

- **v0.1 — done.** Fetch + role filter + location-lock filter + dedupe + Markdown digest (the existing Go script).
- **v0.2.** Add 24h freshness filter (F2) + external config file (replace hardcoded lists).
- **v0.3.** Add Discord delivery (F7).
- **v0.4.** Add Gmail delivery (F8) + cross-day dedupe via `seen.json` (F6).
- **v0.5.** Toggles wired to config, README, worldwide ranking polish (F10).
- **v1.0.** Multiple role profiles, multiple recipients, clean docs.
- **v2.0 — AI tier.** Lite CV keyword extraction (F11), then semantic CV match + summaries (F12–F13), all off by default.

## 12. Success metrics

- Daily review time drops from ~60 min to **under 5 min**.
- **Zero** location-locked roles appear in the digest.
- **Zero** duplicates across days.
- Runs hands-off for a week with no manual intervention.
- Cost stays at **$0/month**.
