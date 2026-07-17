# Auth MVP — TDD Evidence

Source: user journeys and requirements supplied in this task.

## User journeys

- As a visitor, I can register an active base-user account after accepting the terms.
- As an active user, I can log in and receive a JWT containing approved roles only.
- As an authenticated user, I can request borrower, lender, or verifier; I cannot request admin.
- As an authenticated user, I can retrieve my current profile and approved roles.

## Evidence

| Guarantee | Test | Result |
|---|---|---|
| Registration requires terms acceptance | `TestRegisterRejectsUnacceptedTerms` | PASS |
| Registration normalizes email and hashes the password | `TestRegisterHashesPasswordAndNormalizesEmail` | PASS |
| Existing email is rejected | `TestRegisterRejectsExistingEmail` | PASS |
| Login token carries approved roles | `TestLoginIssuesTokenWithApprovedRoles` | PASS |
| Incorrect passwords and inactive users cannot log in | `TestLoginRejectsIncorrectPassword`, `TestLoginRejectsInactiveUser` | PASS |
| Blocked accounts cannot view a profile or submit role requests | `TestProfileRejectsBlockedUser`, `TestRequestRoleRejectsBlockedUser` | PASS |
| Admin role requests and duplicate open requests are rejected | `TestRequestRoleRejectsAdmin`, `TestRequestRoleRejectsOpenDuplicate` | PASS |
| Valid borrower request and current role profile are returned | `TestRequestRoleCreatesBorrowerRequest`, `TestProfileReturnsCurrentApprovedRoles` | PASS |
| JSON responses never expose a password hash | `TestUserJSONNeverExposesPasswordHash` | PASS |

## Commands and output

- RED: `go test ./internal/auth/service` failed because the auth service did not exist.
- GREEN: `go test ./...` passed.
- Build: `go build ./cmd/api` passed.
- Coverage: `go test -cover ./internal/auth/service` reported `coverage: 81.2% of statements`.

## Known MVP boundary

The endpoint records a submitted role request. KTP upload/verification, borrower/lender/verifier evidence, admin review transitions, and conflict-of-interest declarations are follow-up workflow modules.
