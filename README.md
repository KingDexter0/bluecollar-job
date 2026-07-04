# Blue-Collar Job Matchmaking & Automated Onboarding Platform

Backend foundation for a blue-collar hiring platform that will support worker onboarding, verification tiers, employer job posting, applicant tracking, interview scheduling, referrals, and subscriptions.

## Tech Stack

- Go
- Gin HTTP framework
- PostgreSQL
- Redis
- Docker Compose
- Next.js
- TypeScript
- Tailwind CSS

## Project Structure

- `cmd/api`: API entrypoint
- `internal/config`: environment configuration
- `internal/database`: PostgreSQL setup
- `internal/cache`: Redis setup
- `internal/models`: domain models
- `internal/repository`: persistence layer
- `internal/service`: business logic
- `internal/handler`: HTTP handlers
- `internal/middleware`: request middleware
- `frontend`: Next.js employer dashboard and local demo UI
- `000001_init_schema.*.sql`: initial PostgreSQL migration files
- `docs`: project architecture notes
- `docs_api.md`: REST API curl examples

## Environment Variables

Copy the example file and fill secrets locally:

```bash
cp .env.example .env
```

Required variables:

- `APP_PORT`: API port, defaults to `8080`
- `DATABASE_URL`: PostgreSQL connection URL
- `REDIS_ADDR`: Redis host and port
- `REDIS_PASSWORD`: Redis password, empty for local development
- `REDIS_DB`: Redis database index
- `JWT_SECRET`: secret used later for signed auth tokens
- `WHATSAPP_VERIFY_TOKEN`: reserved for future WhatsApp webhook verification
- `WHATSAPP_ACCESS_TOKEN`: reserved for future WhatsApp API access
- `AADHAAR_GATEWAY_PROVIDER`: Aadhaar verification provider, defaults to `mock`
- `AADHAAR_GATEWAY_BASE_URL`: authorized Aadhaar/e-KYC gateway base URL
- `AADHAAR_GATEWAY_CLIENT_ID`: authorized gateway client id
- `AADHAAR_GATEWAY_CLIENT_SECRET`: authorized gateway client secret

## Run Locally

Start PostgreSQL and Redis yourself, then run:

```bash
go mod download
go run ./cmd/api
```

The API starts on `http://localhost:8080` unless `APP_PORT` is changed.

## Run With Docker Compose

Create `.env` first:

```bash
cp .env.example .env
```

Then start the full local stack:

```bash
docker compose up --build
```

Docker Compose starts:

- Go API
- PostgreSQL
- Redis

The initial migration SQL is mounted into Postgres and runs when the database volume is first created.

## Frontend Website And Dashboard

The demo frontend lives in `frontend` and connects to the Go API through `NEXT_PUBLIC_API_BASE_URL`.

Create the frontend environment file:

```bash
cd frontend
cp .env.example .env.local
```

Default local value:

```bash
NEXT_PUBLIC_API_BASE_URL=http://localhost:8081
```

Install and run:

```bash
npm install
npm run dev
```

Open `http://localhost:3000`.

Build check:

```bash
npm run build
```

Implemented frontend routes:

- `/`: public landing page
- `/employer/register`: employer registration
- `/employer/login`: employer login
- `/employer/dashboard`: employer summary dashboard
- `/employer/jobs`: employer job management
- `/employer/applications`: ATS, filters, status updates, and interview scheduling
- `/worker/demo`: local worker onboarding, verification, job browsing, and application demo
- `/dev/notifications`: local notification worker preview

Recommended demo flow:

1. Start backend: `docker compose up --build`
2. Start frontend from `frontend`: `npm run dev`
3. Register an employer
4. Create a job
5. Create a worker from `/worker/demo`
6. Apply the worker to the job
7. View the application in `/employer/applications`
8. Update status or schedule an interview

## Database Migrations

Local development uses Postgres container initialization:

```bash
docker compose up --build
```

The file `000001_init_schema.up.sql` is mounted into `/docker-entrypoint-initdb.d/` and runs automatically when the `postgres_data` volume is first created.

To reset the local database and rerun migrations plus seed data:

```bash
docker compose down -v
docker compose up --build
```

To inspect seeded tables:

```bash
docker compose exec postgres psql -U bluecollar -d bluecollarjob -c "\dt"
docker compose exec postgres psql -U bluecollar -d bluecollarjob -c "SELECT COUNT(*) FROM users;"
```

There is no separate seed command yet. Local seed data lives at the bottom of `000001_init_schema.up.sql` and is inserted automatically during a clean Postgres volume initialization. For local development, the reset command above is the migration plus seed command.

The Day 2 schema includes:

- `users`: worker identities, phone numbers, referral codes, target roles, preferred zones, and verification tier
- `worker_identity_verifications`: Aadhaar OTP, document upload, or skipped verification metadata
- `employers`: employer accounts with unique email
- `jobs`: employer-owned job posts
- `applications`: worker applications tied to users, jobs, and employers
- `interview_slots`: scheduler slots and slot state
- `referrals`: referral relationships between workers
- `referral_transactions`: cashback transaction tracking
- `subscriptions`: employer subscription tier state
- `notification_events`: queued notification records for future WhatsApp/SMS/email processing

Seed data is included in `000001_init_schema.up.sql`:

- 1 Low risk worker with a mock Aadhaar OTP verified record
- 1 Medium risk worker with a mock secure document reference
- 1 High risk worker who skipped verification
- 1 employer
- 2 jobs
- 2 applications

## Aadhaar Verification Design

The database never stores raw Aadhaar numbers or OTP values. The `worker_identity_verifications` table stores only:

- `aadhaar_last4`
- `aadhaar_masked`
- `aadhaar_hash`
- `aadhaar_reference_key`
- `document_ref`
- consent and verification timestamps

Document uploads should store only secure object storage references such as an S3/GCS/Azure Blob object key. Raw document binaries do not belong in PostgreSQL.

Verification outcomes map to worker risk tiers:

- Aadhaar OTP verified through an authorized gateway: `Low`
- Aadhaar-linked mobile unavailable or OTP incomplete, followed by document upload: `Medium`
- Verification skipped: `High`

The Go code includes an `AadhaarGateway` interface and a local `MockAadhaarGateway`. Real Aadhaar/e-KYC integrations must use an authorized provider, and credentials must come only from environment variables. Do not log Aadhaar numbers, OTPs, gateway secrets, or raw identity documents.

## Redis Aadhaar OTP Keys

OTP state is intentionally ephemeral and belongs in Redis, not PostgreSQL.

Recommended key design:

- OTP transaction key: `aadhaar_otp:{phone_number}`
- Retry counter: `aadhaar_otp_retry:{phone_number}`
- Rate limit key: `rate_limit:aadhaar_otp:{phone_number}`
- TTL: 5 minutes for OTP transaction state
- Store only a hashed OTP or authorized gateway transaction id
- Track retry count and stop verification after the configured maximum
- Rate limit by phone number and by IP/device where available

Never store raw OTP values permanently and never write OTPs to logs.

## Repository Layer

The repository layer lives in `internal/repository` and exposes interfaces for:

- `UserRepository`
- `IdentityVerificationRepository`
- `EmployerRepository`
- `JobRepository`
- `ApplicationRepository`

PostgreSQL implementations use `context.Context` and parameterized SQL through `pgx`. Aadhaar verification repositories only persist masked Aadhaar metadata, hash/reference fields, consent timestamps, and document object references.

Run the normal package checks:

```bash
go test ./...
go build ./...
```

Run repository integration tests against the Docker PostgreSQL database:

```bash
$env:TEST_DATABASE_URL="postgres://bluecollar:bluecollar@localhost:5432/bluecollarjob?sslmode=disable"
go test ./internal/repository -run TestPostgresRepositories -count=1 -v
```

If `TEST_DATABASE_URL` is not set, the integration test skips safely.

## Employer Auth And Jobs

Employer registration and login return JWT bearer tokens. Protected employer routes require:

```bash
Authorization: Bearer {jwt_token}
```

Implemented employer routes:

- `POST /api/v1/employers/register`
- `POST /api/v1/employers/login`
- `GET /api/v1/employers/me`
- `PATCH /api/v1/employers/me`
- `POST /api/v1/employer/jobs`
- `GET /api/v1/employer/jobs`
- `GET /api/v1/employer/jobs/:id`
- `PATCH /api/v1/employer/jobs/:id`
- `PATCH /api/v1/employer/jobs/:id/status`

Growth tier employers can have a maximum of 7 active jobs. Enterprise tier employers can have unlimited active jobs.

## Health Check

```bash
curl http://localhost:8080/health
```

Example healthy response:

```json
{
  "status": "ok",
  "postgres": {
    "status": "ok"
  },
  "redis": {
    "status": "ok"
  },
  "checked_at": "2026-06-19T06:00:00Z"
}
```

If PostgreSQL or Redis is unavailable, the endpoint returns `503` with the unavailable component marked in the response.

Additional production probes:

```bash
curl http://localhost:8080/live
curl http://localhost:8080/ready
curl http://localhost:8080/metrics
```

## Production Readiness

See `docs_production.md` for production environment variables, migration commands, Docker production builds, Kubernetes/Linode LKE deployment notes, SSL setup, security checklist, backup checklist, rollback checklist, and smoke testing.

Migration commands:

```bash
go run ./cmd/migrate status
go run ./cmd/migrate up
go run ./cmd/migrate down
```

Production-like Docker build:

```bash
docker compose -f docker-compose.prod.yml build
```
