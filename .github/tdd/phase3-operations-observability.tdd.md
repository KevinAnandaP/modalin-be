# Phase 3 — operations and observability

Source: Phase 3 of the production-hardening plan, derived in the Codex task on 2026-07-25.

## User journeys

- As an operator, I can distinguish a live API process from an API ready to serve database-backed traffic.
- As an operator, I receive request counters and structured logs without credentials or request payloads.
- As a deployer, a termination signal stops the scheduler, HTTP server, and database connection in a controlled order.

## RED → GREEN evidence

| Behavior | RED evidence | GREEN evidence | Guarantee |
| --- | --- | --- | --- |
| Safe HTTP configuration and errors | `go test ./cmd/api` failed because Fiber used its default 4 MB body limit and had no configured timeouts. | `go test ./cmd/api` passed. | API enforces 6 MB request bodies, read/write/idle timeouts, emits a request ID, and redacts internal 5xx error messages. |
| Liveness and readiness | `go test ./internal/health/handler` failed because `Live` and `Ready` did not exist. | Focused health tests passed. | `/health` is process liveness; `/ready` returns 503 when PostgreSQL is unavailable. |
| Metrics | `go test ./pkg/middleware` failed because `NewRequestMetrics` did not exist. | Middleware tests passed. | Aggregate request count, latency, and HTTP status counters are collected without paths, identities, headers, or bodies. |
| Graceful lifecycle and database logging | Focused tests failed because `waitForNextRun`, `databaseLogMode`, and `shutdownApp` did not exist. | Job/database/API tests passed. | `SIGTERM` context cancellation stops the scheduler; Fiber shuts down before database close; production GORM logging uses Warn rather than per-query Info logs. |

## Verification

- `go test ./...` — PASS.
- `go vet ./...` — PASS.
- `git diff --check` — PASS.
- Focused coverage: middleware 85.6%, API 45.0%, health handler 50.0%.
- `docker compose build app migrate` — attempted after the changes, but Docker BuildKit failed with a local `containerd-overlayfs/metadata_v2.db` input/output error while committing a build layer. Go tests compile both commands; rebuild after repairing local Docker storage.

## Operational endpoints

- `GET /health`: liveness, no database check.
- `GET /ready`: readiness, checks PostgreSQL.
- `GET /metrics`: aggregate JSON counters only. Put this endpoint behind monitoring-network access at the reverse proxy before an internet-facing production deployment.
