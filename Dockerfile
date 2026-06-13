# syntax=docker/dockerfile:1

# ---- Build stage ----------------------------------------------------------
# Static Go build (CGO disabled) plus the goose migration tool, which the
# entrypoint runs against Postgres before the API serves traffic.
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache module downloads separately from the source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

# goose CLI (same version the Makefile pins) — copied into the runtime image.
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.22.1

COPY . .

# Fully static binary so it runs on a minimal base with no libc surprises.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/postal ./cmd/postal

# ---- Runtime stage --------------------------------------------------------
FROM alpine:3.20

# ca-certificates: outbound HTTPS to X/Instagram/TikTok/Stripe/Paystack/R2.
# tzdata: correct scheduling timestamps. wget: container healthcheck.
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S postal && adduser -S -G postal postal

WORKDIR /app

COPY --from=build /out/postal /app/postal
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY db/migrations /app/db/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

USER postal

# API role listens here; the worker role ignores it. Overridable via env.
ENV POSTAL_HTTP_ADDR=:8080
EXPOSE 8080

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["serve"]
