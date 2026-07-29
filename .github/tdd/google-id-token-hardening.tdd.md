# Google ID-token hardening — TDD evidence

Source: Phase 1 of the production-hardening plan, derived in the Codex task on 2026-07-25.

## User journeys

- As a Google user, I can authenticate only with a Google-signed ID token for an allowed OAuth client, so a caller cannot forge an identity by submitting an email or Google subject.
- As an existing password user, I am not silently linked to a submitted Google identity, so account linking requires a future authenticated re-authentication flow.

## RED → GREEN evidence

| Behavior | RED evidence | GREEN evidence | Guarantee |
| --- | --- | --- | --- |
| Verified Google identity contract | `go test ./internal/auth/service` failed to compile because `GoogleIdentity`, `NewAuthServiceWithGoogleVerifier`, and `ErrGoogleAccountLinkRequired` did not exist. | `go test ./internal/auth/service ./internal/auth/handler ./pkg/config ./internal/router` passed. | Authentication receives an ID token and derives identity only through a verifier. |
| Google signature and claims | Tests for a signed fixture and a wrong audience were added before `GoogleIDTokenVerifier`. | `go test ./internal/auth/service` passed. | Only RS256 Google JWKS signatures with an allowed audience, Google issuer, non-expired token, subject, email, and verified email are accepted. |
| Unsafe automatic linking | Test `TestGoogleAuthRejectsExistingEmailWithoutExplicitLink` was added before the service change. | `go test ./internal/auth/service` passed. | A matching email without the same verified Google subject returns a link-required conflict and does not issue a JWT. |

## Verification performed

- `go test ./...` — PASS.
- `go vet ./...` — PASS.
- `docker compose build app migrate` — PASS; local images were inspected as `modalin-be-app:latest` and `modalin-be-migrate:latest`.
- `go test -cover ./internal/auth/service ./pkg/config` — auth service 69.3%, config 93.3%.

## Coverage and known gaps

The new cryptographic verifier's successful and wrong-audience paths are covered by locally generated RSA/JWKS fixtures. The existing auth-service package remains below the repository target because older role-review and temporary-token error branches predate this change. No live Google token, credentials, production database, or production endpoint was used.

## Follow-up

Set `GOOGLE_OAUTH_CLIENT_IDS` to the real web/mobile OAuth client IDs before production. A separate authenticated re-authentication endpoint is required before offering Google account linking for existing password accounts.
