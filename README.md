# wdns

Minimal Docker image that contains the compiled `wdns` Go binary. The service invokes an external `kdig` binary from the `internal/resolver` package to perform DNS queries (no runtime Go DNS client). It supports multiple transports like UDP, TCP, TLS (DoT) and HTTPS (DoH) and returns structured JSON output.

**Build**:

```bash
docker build -t wdns:latest .
```

**Run**:

```bash
docker run --rm -it wdns:latest
```

Notes:

- The `Dockerfile` is multi-stage: it builds the binary using `golang:1.25.7-trixie` then copies the result into a slim `debian:trixie-slim` image.
- The service performs DNS queries by invoking the external `kdig` binary (installed in the runtime image).

## API

The service exposes a single HTTP endpoint:

- `POST /query` run a DNS query and return structured output.

```bash
❯ curl -s -X POST http://localhost:8080/query  -H 'Content-Type: application/json'  -d '{"nameserver":"9.9.9.9","name":"example.com","type":"AAAA","transport":"tls"}' | jq
{
  "status": 200,
  "success": true,
  "timestamp": "2026-02-05T11:22:52Z",
  "request": {
    "nameserver": "9.9.9.9",
    "short": false,
    "dnssec": false,
    "type": "AAAA",
    "transport": "tls",
    "name": "example.com",
    "json": false
  },
  "command": "kdig @9.9.9.9 example.com AAAA +tls",
  "answer": ";; TLS session (TLS1.3)-(ECDHE-X25519)-(ECDSA-SECP256R1-SHA256)-(AES-256-GCM)\n;; ->>HEADER<<- opcode: QUERY; status: NOERROR; id: 42033\n;; Flags: qr rd ra ad; QUERY: 1; ANSWER: 2; AUTHORITY: 0; ADDITIONAL: 1\n\n;; EDNS PSEUDOSECTION:\n;; Version: 0; flags: ; UDP size: 512 B; ext-rcode: NOERROR\n\n;; QUESTION SECTION:\n;; example.com.        \t\tIN\tAAAA\n\n;; ANSWER SECTION:\nexample.com.        \t25\tIN\tAAAA\t2606:4700::6812:1b78\nexample.com.        \t25\tIN\tAAAA\t2606:4700::6812:1a78\n\n;; Received 96 B\n;; Time 2026-02-05 11:22:52 UTC\n;; From 9.9.9.9@853(TLS) in 67.2 ms\n"
}
```

Request JSON fields:

- `nameserver` (string, required): DNS server to query (e.g. `1.1.1.1`).
- `name` (string, required): domain name to query (e.g. `example.com`).
- `type` (string, required): record type either `A` or `AAAA`.
- `transport` (string, optional): transport to use for the query. Allowed values: `tcp`, `tls`, `https`, or empty (UDP). The service uses the chosen transport when performing the DNS query.
- `short` (bool, optional): when true, return compact output.
- `json` (bool, optional): when true, return structured JSON for the answer field when possible.
- `dnssec` (bool, optional): when true, the service sets the EDNS0 DO bit requesting DNSSEC-related records (RRSIGs) from the upstream server. Default is `false`. Note: `wdns` will request DNSSEC records but does not perform cryptographic validation of signatures.

Response additions:

- `command` (string): a kdig-equivalent command that represents the DNS query executed by the service. Useful for debugging and reproducing queries locally.

Example request (curl):

```bash
curl -s -X POST http://localhost:8080/query \
 -H 'Content-Type: application/json' \
 -d '{"nameserver":"1.1.1.1","name":"example.com","type":"A","transport":"tcp","short":true}'
```

Example request with DNSSEC enabled (EDNS0 DO bit set):

```bash
curl -s -X POST http://localhost:8080/query \
 -H 'Content-Type: application/json' \
 -d '{"nameserver":"1.1.1.1","name":"example.com","type":"A","transport":"tcp","short":true,"dnssec":true}'
```

Example successful response:

```json
{
 "status":200,
 "success":true,
 "timestamp":"2026-02-04T12:00:00Z",
 "request":{ "nameserver":"1.1.1.1","name":"example.com","type":"A","transport":"tcp","short":true },
 "command":"kdig @1.1.1.1 example.com A +tcp",
 "answer":"93.184.216.34"
}
```

On error, the response will include `success:false` and an `error` string with details.

## `wdns` binary / functionality

The `wdns` Go program is an HTTP service that performs DNS queries by invoking the external `kdig` binary via the bundled `internal/resolver` implementation. Key behavior:

- Constructs a `kdig` command according to the request parameters and executes it.
- Returns the command's output in `answer` (parsed JSON when `json=true`).

Run locally (without Docker):

```bash
go run ./cmd/wdns
```

In the Docker image the compiled binary is installed at `/usr/local/bin/wdns` and the image's `ENTRYPOINT` runs it directly.

## Security & image notes

- The provided `Dockerfile` builds the `wdns` binary and produces a minimal runtime image based on `debian:trixie-slim`.
- Runtime image runs as a non-root numeric UID (1000) and the binary is owned by that user.
- The `docker-compose.yml` runs the container with a read-only root filesystem and mounts a `tmpfs` at `/tmp` to limit writable surface.
- For a smaller production image, we can produce a static Go binary and use a `distroless` or `scratch` runtime.

## Rate limiting

- The HTTP handler implements a simple per-client (per IP) token-bucket rate limiter.
- Configure via environment variables:
  - `RATE_LIMIT_RPS` requests per second (default `10`).
  - `RATE_LIMIT_BURST` burst capacity (default `20`).
- If a client exceeds the configured rate, the service responds with HTTP `429 Too Many Requests` and a `Retry-After` header.

## Environment variables

- `PORT` port the server listens on (default `8080`).
- `RATE_LIMIT_RPS` requests per second (default `10`).
- `RATE_LIMIT_BURST` burst capacity (default `20`).
- `TRUSTED_PROXIES` comma-separated CIDRs of proxies trusted to set forwarding headers (example: `10.0.0.0/8,192.168.0.0/16`). When set, the service will extract the client IP from `X-Forwarded-For` / `X-Real-IP` headers for rate-limiting. SECURITY: only set when running behind a trusted reverse proxy; headers can be spoofed by clients.

## Production compose example

A minimal `docker-compose.yml` for production-like runs with resource limits and healthcheck:

```yaml
version: "3.9"
services:
  wdns:
    build: .
    image: wdns:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - RATE_LIMIT_RPS=10
      - RATE_LIMIT_BURST=20
      - TRUSTED_PROXIES=10.0.0.0/8,192.168.0.0/16
    deploy:
      resources:
        limits:
          cpus: '0.25'
          memory: 50M
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:8080/ || exit 1"]
      interval: 30s
      timeout: 5s
      retries: 3
    user: "1000:1000"
    read_only: true
    tmpfs:
      - /tmp:exec,mode=1777
```

Notes:

- The service runs as non-root (`UID 1000`) and the root filesystem is set read-only in compose; `tmpfs` is used for writable paths like `/tmp`.
- Adjust `RATE_LIMIT_RPS` and `RATE_LIMIT_BURST` for your traffic profile.

## docker-compose notes

Use `docker compose up --build` to build and run the service. The compose file configures resource limits, healthcheck, runs as UID `1000`, and sets the container root filesystem to read-only with writable `/tmp`.

## CI

This repository includes a GitHub Actions workflow at `.github/workflows/ci.yml` that performs the following steps on push and pull requests:

- Run `golangci-lint run ./...`.
- Run `go test ./... -v`.
- Build the Docker image.
- Scan the built image with Trivy.

Locally you can run the same checks using the Makefile and go tooling:

```bash
make lint        # runs golangci-lint
go test ./... -v
docker build -t wdns:local .
docker run --rm -p 8080:8080 wdns:local
```
