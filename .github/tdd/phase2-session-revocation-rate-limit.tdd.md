# Phase 2 — session revocation and authentication rate limiting

Source: Phase 2 of the production-hardening plan, derived in the Codex task on 2026-07-25.

## User journeys

- As an administrator, I need blocked users and removed roles to lose API access immediately, rather than at JWT expiry.
- As an API operator, I need public authentication endpoints to throttle repeated attempts before they reach password or Google-token verification.

## RED → GREEN evidence

| Behavior | RED evidence | GREEN evidence | Guarantee |
| --- | --- | --- | --- |
| Persisted authorization state | `go test ./pkg/middleware` failed because `JWTProtected` accepted only one argument. | `go test ./pkg/middleware` passed. | Protected routes compare JWT token version with the persisted user, reject blocked users, and use current approved roles rather than JWT role claims. |
| Token-version session contract | `go test ./internal/auth/service` failed because `User.TokenVersion` and `AuthService.GetAuthorizationState` did not exist. | Focused service and migration tests passed. | Fresh login tokens carry a positive token version; the database schema has a reversible `token_version` migration. |
| Public auth rate limiting | `go test ./pkg/middleware` failed because `LoginRateLimit` did not exist. | Middleware tests passed. | Login allows 10 attempts/minute per IP/email; registration allows 5/minute per IP; Google auth allows 10/minute per IP, then returns HTTP 429. |

## Verification

- `go test ./...` — PASS.
- `go vet ./...` — PASS.
- Focused: `go test ./internal/auth/service ./pkg/middleware ./internal/router ./pkg/database ./internal/model` — PASS.
- `git diff --check` — PASS.
- `docker compose build app migrate` — PASS.

## Operational boundary

The limiter is intentionally in-memory for the single-instance demo deployment. Before horizontal scaling, configure Fiber with a shared Redis-compatible limiter store. The persisted authorization lookup runs per protected request; introduce a short, explicit cache only after measuring its database impact, while preserving immediate invalidation behavior.

Focused coverage: middleware 83.9%, auth service 69.6%, configuration 93.3%. The auth-service package remains below 80% because several older role-review and temporary-token branches are not covered; the new persisted-session and Google-token paths have dedicated tests.
