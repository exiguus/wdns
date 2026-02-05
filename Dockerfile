# Multi-stage Dockerfile
# Builder: compile the Go binary
FROM golang:1.25.7-trixie AS builder
WORKDIR /src
COPY . .
RUN go mod download

# Build a static, optimized Go binary (CGO disabled)
## Ensure a `cmd/wdns` entrypoint exists in case CI checkout omits it.
# This creates a tiny `main` that calls the package `wdns.Run()` so the
# builder can produce a runnable binary without requiring `cmd/` in the repo.
RUN mkdir -p /src/cmd/wdns \
    && cat > /src/cmd/wdns/main.go <<'EOF'
package main

import "github.com/exiguus/wdns"

func main() {
    wdns.Run()
}
EOF

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /wdns ./cmd/wdns
RUN chmod +x /wdns || true

### Certs stage: extract CA bundle for TLS (DoH/DOT)
FROM debian:trixie-slim AS certs
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

### Final image: debian-based runtime with kdig installed
FROM debian:trixie-slim AS runtime

# Install kdig (knot dnsutils) and CA certificates for DoH/TLS support
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates knot-dnsutils curl \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /wdns /usr/local/bin/wdns

# Ensure TLS libs in Go find the CA bundle
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

# Default runtime environment variables (can be overridden at container runtime)
ENV PORT=8080
ENV RATE_LIMIT_RPS=20
ENV RATE_LIMIT_BURST=200
ENV TRUSTED_PROXIES=

# Run as non-root numeric UID (no passwd file required)
USER 1000
ENTRYPOINT ["/usr/local/bin/wdns"]

# Docker healthcheck: probe the internal HTTP health endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f "http://localhost:${PORT:-8080}/healthz" || exit 1
