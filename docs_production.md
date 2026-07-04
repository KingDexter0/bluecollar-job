# Production Readiness

This project is prepared for production-style deployment without real WhatsApp, Aadhaar/e-KYC, or payout provider credentials.

## Environment Modes

`APP_ENV` must be one of `local`, `development`, `staging`, or `production`.

Dev-only APIs under `/api/v1/dev/*` are registered only in `local` and `development`.

## Required Production Variables

Production and staging fail fast if these are missing or weak:

- `APP_ENV`
- `APP_PORT`
- `DATABASE_URL`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `ADMIN_TOKEN`
- `CORS_ALLOWED_ORIGINS`
- `FRONTEND_URL`
- `WHATSAPP_VERIFY_TOKEN`
- `OBJECT_STORAGE_BUCKET` when `DOCUMENT_UPLOAD_ENABLED=true`

`JWT_SECRET` must be at least 32 characters in staging/production and must not use placeholder values.
`ADMIN_TOKEN` must also be at least 32 characters in staging/production.

## Migrations

```bash
go run ./cmd/migrate status
go run ./cmd/migrate up
go run ./cmd/migrate down
go run ./cmd/migrate baseline
```

The command uses `MIGRATIONS_DIR` if set and falls back to the repository root for the existing `000001_init_schema.*.sql` files.

Use `baseline` only when an existing database already has the expected production tables and only needs the `schema_migrations` record. It verifies required tables before marking `000001_init_schema` as applied. In staging/production it is blocked unless `ALLOW_PRODUCTION_BASELINE=true` is set for that one command.

## Production Docker

```bash
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d
```

## Health And Metrics

- `/live`: API process liveness.
- `/ready`: API, PostgreSQL, and Redis readiness.
- `/health`: full local health response.
- `/metrics`: minimal Prometheus-style request counter.

## Auth Hardening Plan

Employer JWT expiry is configurable through `JWT_TTL_HOURS`.

Refresh tokens are intentionally deferred. Production plan:

- Store hashed refresh tokens in PostgreSQL.
- Rotate refresh tokens on every refresh.
- Revoke refresh tokens on logout/password reset.
- Add device/session tracking.
- Use short-lived access tokens.

## Object Storage

Document uploads should use object storage and persist only `document_ref` in PostgreSQL.

Current providers:

- `local`: development/mock provider.
- `linode`: placeholder for Linode Object Storage integration.

Production upload policy:

- Store only object keys/references in PostgreSQL.
- Reject unexpected content types.
- Enforce upload size limits before storage.
- Generate signed URLs server-side only when needed.
- Never store raw document binaries in PostgreSQL.

## Provider Interfaces

Real integrations are intentionally not enabled yet. The codebase already keeps provider boundaries clean:

- WhatsApp sender interface with mock sender; add Meta/OpenWA providers behind the same interface.
- Aadhaar gateway interface with mock gateway; add authorized e-KYC provider credentials through env only.
- Referral payout gateway interface with mock payout; add UPI/payout provider behind the gateway.
- Object storage interface with local provider and Linode scaffold.

All real providers must use safe error messages and must never log Aadhaar, OTP, passwords, hashes, or raw document paths.

## PostgreSQL Backup And Restore

Do not run PostgreSQL inside the app container in production. Use managed PostgreSQL or a separately operated database service.

Backup:

```powershell
$env:DATABASE_URL="postgres://user:pass@host:5432/bluecollarjob?sslmode=require"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\backup-postgres.ps1
```

Restore drill:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\restore-postgres.ps1 -BackupPath .\backups\bluecollarjob-YYYYMMDD-HHMMSS.dump
```

Run restore tests against a staging database, never directly against production first.

## Database Index Review

The schema includes indexes for common paths:

- workers by `phone_number`
- workers by `referral_code`
- jobs by `employer_id` and `is_active`
- applications by `user_id`, `job_id`, and `status`
- interview slots by `application_id`
- notifications by `status`
- referrals and referral transactions by user/status access paths

Review slow queries with `EXPLAIN ANALYZE` after real traffic arrives.

## Redis Production Notes

- Require `REDIS_PASSWORD` in staging/production.
- Enable Redis persistence according to the hosting plan, but treat PostgreSQL as the source of truth.
- OTP keys must expire; current OTP services use a 5 minute TTL.
- Conversation state keys must expire; chatbot state writes include TTL.
- Rate-limit keys expire with the configured rate-limit window.

## Kubernetes / Linode LKE

Manifests live under `deploy/k8s`. Replace all placeholder images, hosts, and secrets before deployment.

Apply config, secrets, deployments, services, HPAs, then ingress.

Linode LKE outline:

1. Create an LKE cluster.
2. Install NGINX Ingress Controller.
3. Install cert-manager and configure an issuer.
4. Create managed PostgreSQL/Redis or provision external services.
5. Create Kubernetes Secrets from `deploy/k8s/*secret-template.yaml`.
6. Apply `deploy/k8s/configmap.yaml`.
7. Apply backend/frontend deployments and services.
8. Apply HPAs and ingress.
9. Run `/ready` and the smoke test against the public API URL.

## NGINX Ingress And SSL

`deploy/k8s/ingress.yaml` contains placeholders for API and frontend domains plus a cert-manager issuer annotation. Install NGINX ingress and cert-manager on LKE before applying ingress.

## Security Checklist

- Set `APP_ENV=production`.
- Set explicit `CORS_ALLOWED_ORIGINS`.
- Use a strong `JWT_SECRET`.
- Use a strong `ADMIN_TOKEN` for `/api/v1/admin/*`.
- Keep `.env` and Kubernetes Secret values out of Git.
- Keep dev APIs disabled in production.
- Avoid request body logging on sensitive routes.
- Rotate secrets before go-live.
- Serve API and frontend over HTTPS.

## Backup Checklist

- PostgreSQL daily logical backups.
- Point-in-time recovery where available.
- Redis persistence for recovery convenience only.
- Object storage lifecycle/versioning for documents.
- Restore drills before launch.

## Rollback Checklist

- Keep previous Docker image tags.
- Run migrations separately from app deploys.
- Prefer forward-fix migrations in production.
- Use `kubectl rollout undo deployment/bluecollar-api` for app rollback.

## Smoke Test

```powershell
.\scripts\smoke-test.ps1 -ApiBaseUrl http://localhost:8081
```

Full demo flow:

```powershell
.\scripts\demo-flow.ps1 -ApiBaseUrl http://localhost:8081
```

## Known Limitations

- Real WhatsApp/OpenWA/Meta integration is pending.
- Real Aadhaar/e-KYC integration is pending.
- Real UPI/payout integration is pending.
- Production deployment is prepared but not yet executed.
