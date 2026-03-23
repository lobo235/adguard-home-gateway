# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Correct Vault secret path in deploy spec to use `kv/data/nomad/default/adguard-home-gateway`
- Change `vault.change_mode` from `noop` to `restart` in deploy spec

### Changed
- Docker build workflow resolves version from git tags for non-tag builds

## [v1.1.0] - 2026-03-21

### Added
- `GET /rewrites/{domain}` — single-domain lookup endpoint; returns the rewrite for that domain or 404
- `PUT /rewrites/{domain}` upsert — now creates the entry if it does not exist instead of returning 404; makes the endpoint idempotent for "ensure this DNS entry exists" workflows
- `.golangci.yml` — strict linter config (errcheck, govet, staticcheck, unused, gocyclo, misspell, revive, goimports)
- `.githooks/pre-commit` — runs lint and tests before every commit; activate with `make hooks`
- `make hooks` Makefile target

### Changed
- `DELETE /rewrites/{domain}` — no longer requires `?answer=` query parameter; looks up the existing answer automatically by domain, returns 404 if the domain has no rewrite

## [v1.0.0] - 2026-03-14

### Added
- Initial release
- `GET /health` — unauthenticated health check; pings all AdGuard Home servers, returns version
- `GET /rewrites` — list all DNS rewrite entries across all servers
- `POST /rewrites` — add a DNS rewrite to all servers
- `PUT /rewrites/{domain}` — update an existing rewrite (lookup existing answer, delete, re-add)
- `DELETE /rewrites/{domain}` — delete a rewrite by domain and `?answer=` parameter
- `MultiClient` fan-out: writes go to all replica servers, reads return from first reachable
- Bearer token authentication via `GATEWAY_API_KEY`
- Basic Auth to upstream AdGuard Home via `ADGUARD_USER` / `ADGUARD_PASSWORD`
- TLS skip-verify support via `ADGUARD_TLS_SKIP_VERIFY`
- Configurable URL scheme via `ADGUARD_SCHEME`
- Structured JSON logging via `log/slog` with configurable level
- Nomad job spec with Vault/Consul Template secret injection
- Multi-stage Docker build; images published to `ghcr.io/lobo235/adguard-home-gateway`
- GitHub Actions CI and Docker publish workflows
