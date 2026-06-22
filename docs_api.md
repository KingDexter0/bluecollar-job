# API Examples

Base URL for local Docker Compose:

```bash
http://localhost:8081
```

## Create Worker

```bash
curl -X POST http://localhost:8081/api/v1/workers \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+919876543299",
    "full_name": "Test Worker",
    "language_preference": "hi",
    "target_role": "Electrician Helper",
    "preferred_zone": "Gurugram",
    "referral_code": "TEST299"
  }'
```

## Get Worker

```bash
curl http://localhost:8081/api/v1/workers/{worker_id}
```

## Update Worker Profile

```bash
curl -X PATCH http://localhost:8081/api/v1/workers/{worker_id}/profile \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "Test Worker Updated",
    "language_preference": "en",
    "target_role": "Senior Electrician Helper",
    "preferred_zone": "Delhi NCR"
  }'
```

## Start Aadhaar Mock OTP

The API accepts Aadhaar only for this request. It stores only masked Aadhaar, last 4 digits, hash/reference key, and consent timestamp.

```bash
curl -X POST http://localhost:8081/api/v1/workers/{worker_id}/identity/aadhaar/start \
  -H "Content-Type: application/json" \
  -d '{
    "aadhaar_number": "123456789012",
    "consent_given": true
  }'
```

## Verify Aadhaar Mock OTP

The local mock gateway accepts any valid 4 to 8 digit OTP.

```bash
curl -X POST http://localhost:8081/api/v1/workers/{worker_id}/identity/aadhaar/verify \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "mock-aadhaar-otp-transaction",
    "otp": "123456"
  }'
```

Successful Aadhaar OTP verification updates the worker to `Low` risk.

## Upload Document Reference

Store only a secure object storage reference, not raw document bytes.

```bash
curl -X POST http://localhost:8081/api/v1/workers/{worker_id}/identity/document \
  -H "Content-Type: application/json" \
  -d '{
    "document_ref": "s3://bluecollarjob-local-documents/mock/worker-aadhaar.jpg"
  }'
```

Document upload updates the worker to `Medium` risk.

## Skip Verification

```bash
curl -X POST http://localhost:8081/api/v1/workers/{worker_id}/identity/skip \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Worker chose to skip verification"
  }'
```

Skipped verification updates the worker to `High` risk.

## Get Latest Identity Verification

```bash
curl http://localhost:8081/api/v1/workers/{worker_id}/identity/latest
```

Responses do not expose Aadhaar hash or raw document storage paths.

## List Jobs

```bash
curl "http://localhost:8081/api/v1/jobs?limit=20&offset=0"
```

## Get Job

```bash
curl http://localhost:8081/api/v1/jobs/{job_id}
```

## Create Application

```bash
curl -X POST http://localhost:8081/api/v1/applications \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "{worker_id}",
    "job_id": "{job_id}",
    "source": "api"
  }'
```

## Get Application

```bash
curl http://localhost:8081/api/v1/applications/{application_id}
```

## List Worker Applications

```bash
curl "http://localhost:8081/api/v1/workers/{worker_id}/applications?limit=20&offset=0"
```

## Employer Registration

```bash
curl -X POST http://localhost:8081/api/v1/employers/register \
  -H "Content-Type: application/json" \
  -d '{
    "company_name": "ACME Facilities",
    "contact_name": "Anita Sharma",
    "email": "owner@acmefacilities.example",
    "password": "change-me-strong-password",
    "phone_number": "+911234567899",
    "city": "Gurugram",
    "state": "Haryana"
  }'
```

## Employer Login

```bash
curl -X POST http://localhost:8081/api/v1/employers/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "owner@acmefacilities.example",
    "password": "change-me-strong-password"
  }'
```

## Employer Profile With JWT

```bash
curl http://localhost:8081/api/v1/employers/me \
  -H "Authorization: Bearer {jwt_token}"
```

## Create Employer Job

Growth tier employers can have up to 7 active jobs. Enterprise tier employers can have unlimited active jobs.

```bash
curl -X POST http://localhost:8081/api/v1/employer/jobs \
  -H "Authorization: Bearer {jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Machine Operator",
    "role": "Machine Operator",
    "description": "Operate factory machines and follow supervisor instructions.",
    "skill_category": "Manufacturing",
    "location_city": "Pune",
    "location_state": "Maharashtra",
    "shift_schedule": "Day shift, 9 AM to 6 PM",
    "wage_min_paise": 1800000,
    "wage_max_paise": 2400000,
    "openings": 2,
    "is_active": true
  }'
```

## List Employer Jobs

```bash
curl "http://localhost:8081/api/v1/employer/jobs?limit=20&offset=0" \
  -H "Authorization: Bearer {jwt_token}"
```

## Update Employer Job Status

```bash
curl -X PATCH http://localhost:8081/api/v1/employer/jobs/{job_id}/status \
  -H "Authorization: Bearer {jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "is_active": false
  }'
```

## List Employer ATS Applications

Employers only see applications for their own jobs. Optional filters: `job_id`, `status`, `verification_tier`, `target_role`, `preferred_zone`, `limit`, `offset`.

```bash
curl "http://localhost:8081/api/v1/employer/applications?status=Applied&verification_tier=Low&preferred_zone=Gurugram" \
  -H "Authorization: Bearer {jwt_token}"
```

## List Applications For One Employer Job

```bash
curl "http://localhost:8081/api/v1/employer/jobs/{job_id}/applications?status=Shortlisted" \
  -H "Authorization: Bearer {jwt_token}"
```

## Get Employer Application

```bash
curl http://localhost:8081/api/v1/employer/applications/{application_id} \
  -H "Authorization: Bearer {jwt_token}"
```

## Update Application Status

Valid statuses are `Applied`, `Shortlisted`, `Slot_Selection_Pending`, `Interview_Scheduled`, `Selected`, and `Rejected`. Status changes to `Shortlisted`, `Slot_Selection_Pending`, `Interview_Scheduled`, `Selected`, or `Rejected` create a `notification_events` record with `Pending` status. No WhatsApp message is sent yet.

```bash
curl -X PATCH http://localhost:8081/api/v1/employer/applications/{application_id}/status \
  -H "Authorization: Bearer {jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "Shortlisted"
  }'
```

## Direct Interview Scheduling

Use RFC3339 timestamps for `starts_at` and `ends_at`. This creates a confirmed interview slot and moves the application to `Interview_Scheduled`.

```bash
curl -X POST http://localhost:8081/api/v1/employer/applications/{application_id}/interview/direct \
  -H "Authorization: Bearer {jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "starts_at": "2026-06-25T10:00:00+05:30",
    "ends_at": "2026-06-25T10:30:00+05:30",
    "timezone": "Asia/Kolkata",
    "factory_location": "ACME Factory, Sector 18, Gurugram",
    "google_maps_url": "https://maps.google.com/?q=ACME+Factory+Gurugram"
  }'
```

## Create Worker-Selectable Interview Slots

Employer must create exactly 3 available slots. The application moves to `Slot_Selection_Pending`.

```bash
curl -X POST http://localhost:8081/api/v1/employer/applications/{application_id}/interview/slots \
  -H "Authorization: Bearer {jwt_token}" \
  -H "Content-Type: application/json" \
  -d '{
    "slots": [
      {
        "starts_at": "2026-06-25T10:00:00+05:30",
        "ends_at": "2026-06-25T10:30:00+05:30",
        "timezone": "Asia/Kolkata",
        "factory_location": "ACME Factory, Sector 18, Gurugram",
        "google_maps_url": "https://maps.google.com/?q=ACME+Factory+Gurugram"
      },
      {
        "starts_at": "2026-06-25T11:00:00+05:30",
        "ends_at": "2026-06-25T11:30:00+05:30",
        "timezone": "Asia/Kolkata",
        "factory_location": "ACME Factory, Sector 18, Gurugram",
        "google_maps_url": "https://maps.google.com/?q=ACME+Factory+Gurugram"
      },
      {
        "starts_at": "2026-06-25T12:00:00+05:30",
        "ends_at": "2026-06-25T12:30:00+05:30",
        "timezone": "Asia/Kolkata",
        "factory_location": "ACME Factory, Sector 18, Gurugram",
        "google_maps_url": "https://maps.google.com/?q=ACME+Factory+Gurugram"
      }
    ]
  }'
```

## Worker Selects Interview Slot

The selected slot becomes `Confirmed`, other available slots for the application become `Cancelled`, and the application moves to `Interview_Scheduled`. Duplicate selection of the same slot returns a conflict error.

```bash
curl -X POST http://localhost:8081/api/v1/applications/{application_id}/interview/select-slot \
  -H "Content-Type: application/json" \
  -d '{
    "slot_id": "{interview_slot_id}"
  }'
```
