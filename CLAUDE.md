# adguard-home-gateway

A Go HTTP API server that wraps the AdGuard Home API to manage DNS rewrites.
Companion project to [nomad-gateway](https://github.com/lobo235/nomad-gateway) —
follows identical patterns and conventions.

## Module

`github.com/lobo235/adguard-home-gateway`

## Quick start

```bash
cp .env.example .env
# fill in .env values
go run ./cmd/server
```

## Running tests

```bash
go test ./...
```

## Building

```bash
go build -ldflags "-X main.version=v1.0.0" -o adguard-home-gateway ./cmd/server
```

## Environment variables

| Var                      | Required | Default | Purpose                                                  |
|--------------------------|----------|---------|----------------------------------------------------------|
| ADGUARD_SERVERS          | yes      | —       | Comma-separated list of AdGuard Home servers (host or host:port) |
| ADGUARD_SCHEME           | no       | http    | URL scheme for all servers: http or https                |
| ADGUARD_USER             | no       | —       | AdGuard Home username (omit if auth is disabled)         |
| ADGUARD_PASSWORD         | no       | —       | AdGuard Home password                                    |
| ADGUARD_TLS_SKIP_VERIFY  | no       | false   | Skip TLS cert verification (for self-signed certs)       |
| GATEWAY_API_KEY          | yes      | —       | Bearer token for callers of this API                     |
| PORT                     | no       | 8080    | Listen port                                              |
| LOG_LEVEL                | no       | info    | Log level: debug, info, warn, error                      |

## API routes

| Method | Path                    | Auth?  | Description                        |
|--------|-------------------------|--------|------------------------------------|
| GET    | /health                 | no     | Ping AdGuard, return version       |
| GET    | /rewrites               | Bearer | List all DNS rewrites              |
| POST   | /rewrites               | Bearer | Add a DNS rewrite                  |
| GET    | /rewrites/{domain}      | Bearer | Get rewrite for a domain (404 if none) |
| PUT    | /rewrites/{domain}      | Bearer | Upsert rewrite (create or replace) |
| DELETE | /rewrites/{domain}      | Bearer | Delete rewrite by domain (404 if none) |

### GET /rewrites/{domain}

Returns the rewrite entry for a specific domain. Returns 404 if no rewrite exists.

### PUT /rewrites/{domain}

Upsert — creates the entry if it does not exist, replaces it if it does.
Idempotent for "ensure this DNS entry exists" workflows.

Request body: `{"answer": "new-value"}`

### DELETE /rewrites/{domain}

Looks up the existing answer by domain automatically. Returns 404 if the domain
has no rewrite. No `?answer=` parameter required.

## Architecture

```
cmd/server/main.go          — entry point, wires deps, handles signals
internal/config/config.go   — ENV-based config with validation
internal/adguard/client.go  — AdGuard Home HTTP API wrapper (Basic Auth)
internal/api/server.go      — HTTP server, route registration
internal/api/middleware.go  — bearerAuth + requestLogger
internal/api/handlers.go    — route handlers
internal/api/errors.go      — writeError / writeJSON helpers
internal/api/health.go      — GET /health handler
```

## Versioning

SemVer. Version is embedded at build time via `-ldflags "-X main.version=<ver>"`.
Git tags (`v1.0.0`, etc.) trigger Docker image publishing via GitHub Actions.

## Security rules

- Never commit `.env`, tokens, passwords, or API keys
- Never commit real hostnames, IPs, datacenter names, or node pool names
- Use placeholders in all deploy/config files committed to the repo
