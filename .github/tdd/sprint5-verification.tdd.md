# Sprint 5 verification TDD evidence

Source: approved in-chat Sprint 5 plan.

## User journeys

- A Tier 3/4 borrower requests field verification before publication.
- An independent verifier submits a field report; the admin decides it.
- An eligible community member casts one replaceable vote.
- Catalogue risk reflects assessment data, including repayment changes.

## Evidence

| Guarantee | Test / command | Result |
| --- | --- | --- |
| Weighted risk score uses the approved formula and bands | `go test ./internal/verification/service` | RED: score 77; GREEN: score 78 after nearest-integer rounding |
| Missing risk signals are neutral | `internal/verification/service/service_test.go` | PASS |
| Existing services, handlers, routes, migrations, and Sprint 5 code compile and pass together | `go test ./...` | PASS |
| No whitespace errors in the patch | `git diff --check` | PASS |

## Coverage and gaps

`go test -cover ./...` was executed. The repository has no global coverage threshold and the new verification service has 6.1% statement coverage, so the ECC 80% target is not yet met. The focused score tests cover the calculation contract; HTTP workflow and PostgreSQL transaction tests for assignment, conflict checks, voting, and migrations remain a follow-up needed to meet the target.

## Hardening follow-up

Implemented: explicit `data_limited`/`missing_components`, refresh paths from financial-record changes, protected one-photo upload/download, reassign/cancel lifecycle, and paginated request listing. `go test ./...` passed after these changes. Focused coverage was re-run and remains 6.2% for the service, with no handler/repository coverage; the 80% target therefore remains open.

## Test and backfill follow-up

Lifecycle tests now cover reassignment revoking the prior verifier, cancellation state transitions, and pagination bounds. The migration `20260724_risk_assessment_backfill` marks legacy assessment records as data-limited. Admins can run `POST /api/v1/admin/risk-assessments/backfill` once per environment to recalculate published, funded, and active campaigns. `go test ./...` passed; focused service coverage improved to 19.8%, but remains below the 80% target.
