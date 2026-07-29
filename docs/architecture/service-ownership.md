# Service ownership: `modalin-be`

`modalin-be` owns the campaign lifecycle and its related financial records:

| Domain | Owner | Write boundary |
| --- | --- | --- |
| Authentication and roles | `modalin-be` (current service) | User, role, and role-request routes only |
| Business profile and financial records | `modalin-be` | Borrower-owned business routes only |
| Campaign, funding, disbursement, repayment | `modalin-be` | Campaign service routes only |
| Verification and risk assessment | `modalin-be` | Verification service routes only |
| Disputes, restructuring, audit logs | `modalin-be` (Sprint 6) | Dedicated service API contract; no direct cross-service writes |

For the demonstration deployment, other services in the monorepo must call this service through `api.modalin.my.id`. They must not connect directly to this service's PostgreSQL tables. A shared JWT secret is acceptable only for the single-host demonstration; each service must still validate its own JWT and authorization policy.

## Public boundary

- Only the reverse proxy exposes ports 80 and 443.
- This service binds to `127.0.0.1:8080` on the host.
- PostgreSQL has no host port mapping.
- `/uploads` is never served statically. Proof files are returned only by authorized API endpoints.
