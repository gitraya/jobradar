# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src
# No third-party deps (stdlib only), so there is nothing to `go mod download`.
COPY go.mod ./
COPY . .
# Static binary: no CGO, so it runs on a bare runtime image.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/jobradar .

# ---- runtime stage ----
FROM alpine:3.20
# ca-certificates: required for HTTPS to the job-board APIs.
# tzdata: lets you set TZ if you ever want non-UTC scheduling.
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 jobradar \
 && mkdir -p /app /data \
 && chown jobradar:jobradar /app /data

COPY --from=build /out/jobradar /usr/local/bin/jobradar
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
# Bake the committed config as a default so the image runs without a bind mount
# (handy on Coolify); a compose bind mount over /app/config.yaml still wins.
COPY config.yaml /app/config.yaml
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

WORKDIR /app
USER jobradar

# Config path, and a persistent cross-day cache on the /data volume.
ENV CONFIG=/app/config.yaml \
    SEEN_PATH=/data/seen.json

ENTRYPOINT ["docker-entrypoint.sh"]
