#!/usr/bin/env bash
#
# Local dev runner for JobRadar.
#
# Loads secrets from .env, then runs the pipeline. Defaults to -dry-run so a
# local run never accidentally posts to Discord or sends email. Pass --deliver
# to actually deliver; any extra args are forwarded to the binary.
#
#   ./run.sh                 # dry-run: fetch + filter, print digest, send nothing
#   ./run.sh --deliver       # actually deliver to the enabled channels
#   ./run.sh --config my.yaml --deliver
#
set -euo pipefail

cd "$(dirname "$0")"

ENV_FILE=".env"
CONFIG="config.yaml"
DELIVER=0
EXTRA=()

# Parse our own flags; forward the rest to the binary.
while [[ $# -gt 0 ]]; do
  case "$1" in
    --deliver) DELIVER=1; shift ;;
    --config)  CONFIG="$2"; shift 2 ;;
    --env)     ENV_FILE="$2"; shift 2 ;;
    *)         EXTRA+=("$1"); shift ;;
  esac
done

# Load .env line-by-line rather than `source`-ing it, so values with spaces
# (e.g. a Gmail App Password "abcd efgh ijkl mnop") don't need quoting and are
# never run as commands. Skips blank lines and # comments; strips optional
# surrounding quotes.
load_env() {
  local file="$1" line key val
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# || "$line" != *=* ]] && continue
    key="${line%%=*}"
    val="${line#*=}"
    key="${key#"${key%%[![:space:]]*}"}"   # trim leading space
    key="${key%"${key##*[![:space:]]}"}"   # trim trailing space
    if [[ "$val" == \"*\" || "$val" == \'*\' ]]; then
      val="${val:1:${#val}-2}"
    fi
    export "$key=$val"
  done < "$file"
}

if [[ -f "$ENV_FILE" ]]; then
  echo "loading env from $ENV_FILE"
  load_env "$ENV_FILE"
else
  echo "no $ENV_FILE found; relying on the current environment"
fi

if [[ ! -f "$CONFIG" ]]; then
  echo "config $CONFIG not found — copy config.example.yaml to $CONFIG first" >&2
  exit 1
fi

ARGS=(-config "$CONFIG")
if [[ "$DELIVER" -eq 0 ]]; then
  ARGS+=(-dry-run)
  echo "mode: dry-run (pass --deliver to send for real)"
else
  echo "mode: deliver"
fi

exec go run . "${ARGS[@]}" ${EXTRA[@]+"${EXTRA[@]}"}
