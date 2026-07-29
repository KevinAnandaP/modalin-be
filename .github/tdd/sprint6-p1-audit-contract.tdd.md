# Sprint 6 P1 audit contract — TDD evidence

**Source plan:** Sprint 6 backlog agreed in this task.

## User journeys

- As an administrator, I can retrieve a bounded, filterable audit trail for investigation.
- As an operator, I can correlate an audit event with its HTTP request ID.
- As the system, I preserve audit events as append-only records and reject duplicate request/action/entity audit entries.

## RED/GREEN evidence

| Task | RED evidence | GREEN evidence | Guarantee |
| --- | --- | --- | --- |
| Audit event/request IDs | `go test ./internal/model ./pkg/database ./pkg/audit` failed because `AuditLog` lacked event/request fields and `pkg/audit` did not exist. | Targeted packages and `go test ./...` passed. | Every new event has a UUID and carries a request ID when supplied. |
| Append-only migration | The RED test reported missing `20260725_audit_log_contract`. | `pkg/database` tests passed. | Migration adds event/request indexes, duplicate request-event guard, and PostgreSQL append-only trigger. |
| Admin audit listing | `internal/audit/service/service_test.go` verifies a bounded normalized page. | `go test ./internal/audit/...` passed. | Admin listing caps page size at 100 and supports repository filters. |

## Final verification

- `go test ./...` — PASS.
- `git diff --check` — PASS.
- The actual database migration was not applied to the existing Docker database; apply it through `docker compose --profile migrate run --rm migrate` during the controlled deployment step.
