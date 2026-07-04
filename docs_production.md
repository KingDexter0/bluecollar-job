# Production Readiness

This project is prepared for production-style deployment without real WhatsApp, Aadhaar/e-KYC, or payout provider credentials.

## Environment Modes

`APP_ENV` must be one of `local`, `development`, `staging`, or `production`.

Dev-only APIs under `/api/v1/dev/*` are registered only in `local` and `development`.

## Required Production Variables

Production and staging fail fast if these are missing or weak:

- `APP_ENV`
- `DATABASE_URL`
- `REDIS_ADDR`
- `JWT_SECRET`
- `CORS_ALLOWED_ORIGINS`

`JWT_SECRET` must be at least 32 characters in staging/production and must not use placeholder values.

## Migrations

```bash
go run ./cmd/migrate status
go run ./cmd/migrate up
go run ./cmd/migrate down
```

The command uses `MIGRATIONS_DIR` if set and falls back to the repository root for the existing `000001_init_schema.*.sql` files.

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

## Kubernetes / Linode LKE

Manifests live under `deploy/k8s`. Replace all placeholder images, hosts, and secrets before deployment.

Apply config, secrets, deployments, services, HPAs, then ingress.

## NGINX Ingress And SSL

`deploy/k8s/ingress.yaml` contains placeholders for API and frontend domains plus a cert-manager issuer annotation. Install NGINX ingress and cert-manager on LKE before applying ingress.

## Security Checklist

- Set `APP_ENV=production`.
- Set explicit `CORS_ALLOWED_ORIGINS`.
- Use a strong `JWT_SECRET`.
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
