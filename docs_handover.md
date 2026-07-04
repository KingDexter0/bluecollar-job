# Handover Checklist

GitHub repo:

- https://github.com/KingDexter0/bluecollar-job

## Required Environment Variables

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
- `DOCUMENT_UPLOAD_ENABLED`
- `NEXT_PUBLIC_API_BASE_URL`
- `NEXT_PUBLIC_APP_ENV`

## Local Setup

```bash
docker compose up --build -d
cd frontend
npm install
npm run dev
```

## Production/Staging Commands

```bash
docker compose --env-file .env.staging -f docker-compose.prod.yml build
docker compose --env-file .env.staging -f docker-compose.prod.yml run --rm api /app/migrate up
docker compose --env-file .env.staging -f docker-compose.prod.yml up -d
```

## Migration Commands

Local:

```bash
go run ./cmd/migrate status
go run ./cmd/migrate up
go run ./cmd/migrate baseline
go run ./cmd/migrate down
```

Docker image:

```bash
docker compose --env-file .env.staging -f docker-compose.prod.yml run --rm api /app/migrate status
```

## Backup And Restore

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\backup-postgres.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\restore-postgres.ps1 -BackupPath .\backups\bluecollarjob-YYYYMMDD-HHMMSS.dump
```

## API Docs

- `docs_api.md`
- `docs_production.md`
- `docs_staging_vps.md`

## Frontend Demo Flow

1. Register employer.
2. Login employer.
3. Create job.
4. Create worker in `/worker/demo`.
5. Verify or skip identity.
6. Apply to job.
7. View ATS.
8. Schedule interview.
9. Open `/admin`.
10. Process notifications/referral payouts.

## Smoke Tests

Windows:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 -ApiBaseUrl http://localhost:8081
```

Linux:

```bash
./scripts/smoke-test.sh http://localhost:8081
```

## Pending Real Integrations

- Meta WhatsApp Business API or OpenWA provider.
- Aadhaar/e-KYC provider.
- UPI/payout provider.
- Linode Object Storage credentials and signed upload/download flow.
- Refresh-token based auth sessions.

## Package Lock Note

The current OneDrive workspace can lock `frontend/package-lock.json`. Before final CI hardening:

1. Move the project to `C:\Projects\BLUECOLLARJOB`.
2. Run `cd frontend`.
3. Run `npm install --package-lock-only`.
4. Run `npm run lint`.
5. Run `npm run build`.
6. Commit the updated lockfile.
7. Switch CI and frontend Dockerfile back to `npm ci`.
