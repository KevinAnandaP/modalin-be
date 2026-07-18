# TDD Evidence: business profile and financial proofs

## Source and journeys

Derived during this implementation from the requested acceptance criteria:

1. A borrower may create, read, update, and deactivate only their active business profile.
2. One user may have only one active business, but may create a replacement after deactivation.
3. A borrower may manage only financial records associated with their active business.
4. A borrower may upload a PNG, JPEG, or PDF proof and receive a stored public URL linked to the owned financial record.

## RED/GREEN evidence

| Behavior | RED evidence | GREEN evidence | Guarantee |
|---|---|---|---|
| Replacement business after deactivation | `go test ./internal/business/service -run TestDeactivateBusinessLetsUserCreateANewActiveBusiness -count=1` failed because `Business.Status` and `DeactivateBusiness` did not exist. | Same command passed. | A deactivated business does not prevent the owner from creating one new active business. |
| Financial record update | `go test ./internal/business/service -run TestUpdateFinancialRecordRecalculatesNetAmount -count=1` failed because `UpdateFinancialRecord` did not exist. | Same command passed. | Updating income/expense recalculates `net_amount`. |
| Proof storage | `go test ./internal/business/storage -run TestLocalProofStorageStoresAllowedFileAndReturnsPublicURL -count=1` failed because the storage package had no production code. | Same command passed. | An accepted proof is written beneath `financial-proofs/` and exposes a `/uploads/...` URL. |
| Multipart proof metadata | `go test ./internal/business/handler -run TestBorrowerUploadProofStoresAmountAndOwnership -count=1` failed because the uploaded proof's `amount` was discarded. | Same command passed. | A borrower-only multipart request persists the proof amount together with its stored URL. |

## Final verification

| Command | Result |
|---|---|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./cmd/api` | PASS |
| `docker compose config` | PASS; `uploads_data` is mounted at `/app/uploads` |
| `go test ./internal/business/handler -count=1` | PASS; covers borrower authorization plus owned business and financial-record endpoint flows |

## Coverage and known gaps

`go test -cover ./...` ran successfully, but the repository does not meet the workflow's 80% aggregate target. Current coverage is 64.8% for auth service, 42.3% for business service, 57.7% for business handler, and 52.1% for proof storage; repositories, router, middleware, and database migration have no automated coverage.

The remaining high-value tests are HTTP integration tests for `RequireRole("borrower")`, multipart upload validation/cleanup, and PostgreSQL migration behavior for the partial unique active-business index. No checkpoint commits were made because the worktree already contained unrelated, uncommitted user changes.
