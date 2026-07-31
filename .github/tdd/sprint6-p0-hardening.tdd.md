# Sprint 6 P0 hardening — TDD evidence

**Source plan:** Sprint 6 backlog agreed in this task.

## User journeys

- As a deployer, I want production startup to reject an unsafe JWT, CORS, or partial admin-bootstrap configuration so that the public API is not deployed with an insecure default.
- As a browser user, I want only `modalin.my.id` to be allowed by CORS so that an untrusted site cannot make credentialed API requests.
- As a non-authorized visitor, I cannot fetch a proof file from `/uploads/...` so that proof access remains authorization-gated.
- As an operator, I receive structured request logs with a correlation ID and without authorization data.

## RED/GREEN evidence

| Task | RED evidence | GREEN evidence | Guarantee |
| --- | --- | --- | --- |
| Production configuration validation | `go test ./pkg/config ./pkg/middleware` failed because `Config` had no CORS/bootstrap fields or `Validate` method. | `go test ./cmd/api ./pkg/config ./pkg/middleware` passed. | Production rejects missing/wildcard CORS, short JWT secrets, and partial bootstrap credentials. |
| Restricted CORS | The same RED command failed because `RestrictedCORS` did not exist. | `go test ./pkg/config ./pkg/middleware` passed. | Only configured origins receive the CORS allow-origin header. |
| No public proof serving | `go test ./cmd/api` failed because `newApp` did not exist. | `go test ./cmd/api ./pkg/config ./pkg/middleware` passed. | The application has no static `/uploads` route. |
| Safe JSON logs | `go test ./pkg/middleware` failed because `JSONRequestLogger` did not exist. | `go test ./pkg/middleware ./cmd/api` passed. | Request logs are JSON and contain no authorization field. |

## Test specification

| # | What is guaranteed | Test file | Type | Result |
| --- | --- | --- | --- | --- |
| 1 | Production configuration rejects unsafe values and accepts explicit HTTPS origins | `pkg/config/config_test.go` | unit | PASS |
| 2 | Configured CORS origin is allowed and unknown origin is not | `pkg/middleware/cors_test.go` | middleware integration | PASS |
| 3 | A direct `/uploads/...` request is not served | `cmd/api/main_test.go` | HTTP integration | PASS |
| 4 | Request logs contain request ID, method, path, and no authorization value | `pkg/middleware/observability_test.go` | middleware integration | PASS |

## Final verification

- `go test ./...` — PASS.
- `docker compose config --quiet` with synthetic required values — PASS.
- `git diff --check` — PASS.
- `docker compose build app migrate` — not verified because the local Docker daemon was unavailable (`dockerDesktopLinuxEngine` pipe was not found). Run this on the demo server before deployment.
- Targeted coverage: `cmd/api` 35.3%, `pkg/config` 92.9%, `pkg/middleware` 87.8%. The application-wide 80% threshold is not yet measured; existing service packages are outside this P0 change and have separate tests.
