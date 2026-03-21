# adguard-home-gateway

A lightweight authenticated REST API gateway for managing [AdGuard Home](https://adguard.com/en/adguard-home/overview.html) DNS rewrites. Built as a companion to [nomad-gateway](https://github.com/lobo235/nomad-gateway).

## Overview

Exposes a simple Bearer-authenticated API for creating, listing, updating, and deleting DNS rewrite entries in AdGuard Home. Intended for homelab automation where services need to register or deregister their own DNS entries.

## Requirements

- Go 1.24+
- A running AdGuard Home instance

## Configuration

Copy `.env.example` to `.env` and fill in the values, or set environment variables directly.

| Variable                   | Required | Default | Description                                                         |
|----------------------------|----------|---------|---------------------------------------------------------------------|
| `ADGUARD_SERVERS`          | yes      | —       | Comma-separated AdGuard Home servers, e.g. `192.168.1.1,192.168.1.2:3000` |
| `ADGUARD_SCHEME`           | no       | `http`  | URL scheme for all servers: `http` or `https`                       |
| `ADGUARD_USER`             | no       | —       | AdGuard Home username (omit if authentication is disabled)          |
| `ADGUARD_PASSWORD`         | no       | —       | AdGuard Home password                                               |
| `ADGUARD_TLS_SKIP_VERIFY`  | no       | `false` | Skip TLS certificate verification (for self-signed certs)           |
| `GATEWAY_API_KEY`          | yes      | —       | Bearer token callers must present to use this API                   |
| `PORT`                     | no       | `8080`  | Port this gateway listens on                                        |
| `LOG_LEVEL`                | no       | `info`  | Log verbosity: `debug`, `info`, `warn`, `error`                     |

## Running

```bash
go run ./cmd/server
```

Or with Docker:

```bash
docker run --env-file .env ghcr.io/lobo235/adguard-home-gateway:latest
```

## API

All routes except `GET /health` require `Authorization: Bearer <GATEWAY_API_KEY>`.

### GET /health

Returns the service status and current version. Unauthenticated.

```json
{"status": "ok", "version": "v1.0.0"}
```

Returns `503` if AdGuard Home is unreachable.

### GET /rewrites

Returns all DNS rewrite entries.

```json
[
  {"domain": "svc.example.com", "answer": "192.168.1.10"},
  {"domain": "db.example.com",  "answer": "192.168.1.11"}
]
```

### POST /rewrites

Add a new DNS rewrite.

```json
{"domain": "svc.example.com", "answer": "192.168.1.10"}
```

Returns `201` on success.

### GET /rewrites/{domain}

Returns the rewrite entry for a specific domain. Returns `404` if no rewrite exists for that domain.

```json
{"domain": "svc.example.com", "answer": "192.168.1.10"}
```

### PUT /rewrites/{domain}

Upsert the answer for a domain. If a rewrite exists, it is replaced. If no rewrite exists, one is created. Idempotent — safe to call without checking whether the entry already exists.

```json
{"answer": "192.168.1.20"}
```

Returns `200` with the current entry on success.

### DELETE /rewrites/{domain}

Delete a specific rewrite entry. Looks up the current answer by domain automatically. Returns `404` if no rewrite exists for that domain.

```
DELETE /rewrites/svc.example.com
```

Returns `204` on success.

## Building

```bash
go build -ldflags "-X main.version=v1.0.0" -o adguard-home-gateway ./cmd/server
```

## Testing

```bash
go test ./...
```

## Docker

```bash
docker build --build-arg VERSION=v1.0.0 -t adguard-home-gateway .
```

Images are published to `ghcr.io/lobo235/adguard-home-gateway` on push to `main` and on `v*` tags.

## Deployment

See [deploy/adguard-home-gateway.hcl](deploy/adguard-home-gateway.hcl) for the Nomad job spec. Secrets are sourced from Vault via Consul Template.
