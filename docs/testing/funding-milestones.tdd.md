# Funding and milestone disbursement — TDD evidence

Source: user journeys derived during this implementation.

## User journeys

- As a lender, I can fund a published campaign without exceeding its target, but never my own campaign.
- As a borrower, I can upload a receipt, invoice, photo, or merchant confirmation after a milestone is disbursed.
- As an admin, I can approve, reject, or request a revision for a proof; approval verifies that milestone and releases the next one.

## Evidence

| # | Guarantee | Test | Type | Result |
|---|---|---|---|---|
| 1 | Campaign owner cannot fund their own campaign and no funding can exceed the remaining target. | `TestPledgeRejectsSelfFundingAndOverfunding` | unit | PASS |
| 2 | A fully funded campaign becomes active and creates a proof-required first disbursement. | `TestFullPledgeCreatesFirstDisbursement` | unit | PASS |
| 3 | Approved proof marks its milestone verified and creates the next disbursement. | `TestApprovedProofVerifiesMilestoneAndCreatesNextDisbursement` | unit | PASS |
| 4 | Each pledge executes under one repository transaction and reads the campaign with a lock before funding. | `TestPledgeUsesTransactionAndLocksCampaignBeforeFunding` | unit | PASS |
| 5 | A pending proof can never qualify to unlock the next milestone. | `TestProofQualifiesForNextMilestoneUnlock` | unit | PASS |
| 6 | A borrower, funding lender, and admin can download a proof; an unrelated lender is denied. | `TestProofDownloadEnforcesCampaignAccess` | handler integration | PASS |
| 7 | PostgreSQL rejects duplicate milestone disbursements and active proofs, and rolls back failed transactions. | `TestPostgresConstraintsAndRollback` | PostgreSQL integration | PASS |

For the transaction regression, RED was confirmed with `go test ./internal/campaign/service ./internal/campaign/job`: the new pledge test failed because the service did not invoke a transaction. GREEN: the same command passed after the transaction and row-lock implementation. Final validation: `go test ./...` passed.

## Coverage and known gaps

`go test -cover ./internal/campaign/...` completed successfully. The current focused coverage is 50.4% for campaign service and 23.1% for the now-disabled milestone scheduler package; it does not meet the 80% target. Additional handler multipart and PostgreSQL transaction/integration tests should be added before a production release.

## Database compatibility

Read-only PostgreSQL check on the local `modalin-db` instance found zero duplicate active `disbursements.milestone_id` values and zero duplicate active `fund_usage_proofs.disbursement_id` values. It is safe for the application AutoMigrate step to create the corresponding unique indexes when the application next starts.

The PostgreSQL integration test is opt-in through `TEST_DATABASE_DSN`; it creates and drops only a randomly named schema. The local run passed against the Docker PostgreSQL service.
