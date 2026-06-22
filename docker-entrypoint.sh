#!/bin/sh
# Entrypoint for the JobRadar container.
#
# Two modes:
#   * Run once (default): runs the pipeline and exits. Drive it with the host's
#     cron / systemd timer, or `docker compose run --rm jobradar`.
#   * Daemon (set RUN_AT=HH:MM, UTC): stays up and runs once a day at that time.
#
# Env:
#   CONFIG        path to the config file (default /app/config.yaml)
#   SEEN_PATH     where to persist the cross-day cache (default /data/seen.json)
#   RUN_AT        "HH:MM" (UTC) — enables daemon mode when set
#   RUN_ON_START  "true" to also run immediately on container start (daemon mode)
#   DRY_RUN       "true" to pass -dry-run (fetch + print, deliver nothing)
# Secrets (DISCORD_WEBHOOK_URL, GMAIL_*) are passed through from the environment.
set -eu

CONFIG="${CONFIG:-/app/config.yaml}"
ARGS="-config $CONFIG"
[ -n "${SEEN_PATH:-}" ] && ARGS="$ARGS -seen $SEEN_PATH"
[ "${DRY_RUN:-false}" = "true" ] && ARGS="$ARGS -dry-run"

run() {
  echo "[jobradar] $(date -u '+%Y-%m-%dT%H:%M:%SZ') starting run"
  # shellcheck disable=SC2086
  jobradar $ARGS
  echo "[jobradar] $(date -u '+%Y-%m-%dT%H:%M:%SZ') run finished"
}

if [ -z "${RUN_AT:-}" ]; then
  run
  exit $?
fi

echo "[jobradar] daemon mode: running daily at ${RUN_AT} UTC"
[ "${RUN_ON_START:-false}" = "true" ] && { run || echo "[jobradar] initial run failed"; }

last=""
while true; do
  cur="$(date -u '+%H:%M')"
  if [ "$cur" = "$RUN_AT" ] && [ "$cur" != "$last" ]; then
    run || echo "[jobradar] run failed; will retry at the next schedule"
    last="$cur"
  fi
  sleep 30
done
