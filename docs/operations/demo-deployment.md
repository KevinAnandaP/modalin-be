# Demo deployment runbook

This runbook deploys `modalin-be` on one home server behind Nginx. It is for the competition demonstration, not a high-availability production deployment.

## Before deployment

1. Point `api.modalin.my.id` to the home server's public IP and obtain a TLS certificate.
2. Install the Nginx configuration from `deploy/nginx/api.modalin.my.id.conf`; Nginx is the only public listener.
3. Create server-only environment values. Do not commit them:

```dotenv
POSTGRES_PASSWORD=<long-random-password>
JWT_SECRET=<at-least-32-random-characters>
CORS_ALLOWED_ORIGINS=https://modalin.my.id
```

4. Optionally set `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` only for the first migration. Remove both values immediately after the administrator has been created.

## Deploy

1. Build and start the database: `docker compose up -d db`.
2. Back up PostgreSQL before any schema change.
3. Apply the explicit migration: `docker compose --profile migrate run --rm migrate`.
4. Start the API: `docker compose up -d --build app`.
5. Check `https://api.modalin.my.id/health` and an authenticated API flow from `https://modalin.my.id`.

## Rollback

1. Stop the new API container and start the preceding image version.
2. Restore a database backup only if the applied migration is not backward compatible.
3. Verify `/health`, login, and one authorized proof download before reopening the demo.

## Operational limits

- This is a single-instance deployment. In-memory rate limiting is valid only while there is one API instance.
- Uploads are held on the local Docker volume. Back up `uploads_data` together with PostgreSQL.
- A later multi-instance deployment requires shared object storage and a shared rate-limit store.
