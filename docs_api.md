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

`referral_code` is optional; if omitted, the API generates one. To register through a referral, pass `referred_by_code`:

```bash
curl -X POST http://localhost:8081/api/v1/workers \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+919876543298",
    "full_name": "Referred Worker",
    "language_preference": "hi",
    "target_role": "Machine Operator",
    "preferred_zone": "Pune",
    "referred_by_code": "RAJU100"
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

## Get Worker Referral Code

```bash
curl http://localhost:8081/api/v1/workers/{worker_id}/referral
```

## List Worker Referrals

```bash
curl "http://localhost:8081/api/v1/workers/{worker_id}/referrals?limit=20&offset=0"
```

## List Referral Cashback Transactions

```bash
curl "http://localhost:8081/api/v1/workers/{worker_id}/referral-transactions?limit=20&offset=0"
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

## Dev: List Notifications

Available only when `APP_ENV` is `development`, `local`, or `dev`. Returns safe notification event rows for local inspection. The response includes a message preview, not raw payload data.

```bash
curl "http://localhost:8081/api/v1/dev/notifications?limit=50&offset=0"
```

Optional filters:

```bash
curl "http://localhost:8081/api/v1/dev/notifications?status=Pending&event_type=application_submitted&limit=20&offset=0"
```

Response shape:

```json
{
  "notifications": [
    {
      "id": "notification-id",
      "user_id": "worker-id",
      "worker_id": "worker-id",
      "phone_number": "+919876543210",
      "event_type": "application_submitted",
      "message_preview": "Your application for Machine Operator has been submitted.",
      "status": "Pending",
      "created_at": "2026-07-04T10:00:00Z",
      "updated_at": "2026-07-04T10:00:00Z"
    }
  ]
}
```

Sensitive fields such as Aadhaar data, OTPs, password hashes, Aadhaar hashes, raw document refs, and raw notification payloads are not returned.

## Dev: Process Notifications Once

Available only when `APP_ENV` is `development`, `local`, or `dev`. This claims pending `notification_events`, sends through the mock WhatsApp sender, and marks each event `Sent` or `Failed`.

```bash
curl -X POST http://localhost:8081/api/v1/dev/notifications/process-once \
  -H "Content-Type: application/json" \
  -d '{
    "limit": 10
  }'
```

## Dev: Set Redis Conversation State

Stores state at `wa_state:{phone_number}`.

```bash
curl -X POST http://localhost:8081/api/v1/dev/redis/state \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+919876543210",
    "state": "awaiting_preferred_zone",
    "data": {
      "language": "hi",
      "source": "local-dev"
    },
    "ttl_seconds": 3600
  }'
```

## Dev: Get Redis Conversation State

URL-encode the `+` in the phone number as `%2B`.

```bash
curl "http://localhost:8081/api/v1/dev/redis/state/%2B919876543210"
```

## Dev: Delete Redis Conversation State

```bash
curl -X DELETE "http://localhost:8081/api/v1/dev/redis/state/%2B919876543210"
```

## Dev: Generate Application Status OTP

Stores only a hashed OTP and safe transaction reference at `app_status_otp:{phone_number}` with a 5 minute TTL. The raw OTP is returned only by this local/dev endpoint for testing.

```bash
curl -X POST http://localhost:8081/api/v1/dev/status-otp/generate \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+919876543210"
  }'
```

## Dev: Verify Application Status OTP

```bash
curl -X POST http://localhost:8081/api/v1/dev/status-otp/verify \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "+919876543210",
    "transaction_id": "{transaction_id}",
    "otp": "{otp_for_local_dev}"
  }'
```

## Dev: Process Referral Cashback Payouts

Available only when `APP_ENV` is `development`, `local`, or `dev`. This claims pending referral cashback transactions, processes them through the mock payout gateway, and marks each transaction `Paid` or `Failed`. No real UPI/payment integration is called.

```bash
curl -X POST http://localhost:8081/api/v1/dev/referrals/process-payouts \
  -H "Content-Type: application/json" \
  -d '{
    "limit": 10
  }'
```

## Admin: Summary

Requires `X-Admin-Token`. Local demo token is `local-admin-token`.

```bash
curl http://localhost:8081/api/v1/admin/summary \
  -H "X-Admin-Token: local-admin-token"
```

Returns aggregate counts and analytics only:

- total workers, employers, jobs, applications, referrals, notification events
- pending/failed notifications
- cashback pending/paid/failed
- applications by status
- workers by verification tier
- jobs active/inactive
- referrals by payout status

Sensitive fields such as password hashes, Aadhaar hashes, raw Aadhaar, OTPs, and internal document paths are not returned.

## Admin: Notification List

```bash
curl "http://localhost:8081/api/v1/admin/notifications?status=Pending&event_type=interview_scheduled&limit=20&offset=0" \
  -H "X-Admin-Token: local-admin-token"
```

## Admin: Referral Cashback Transactions

```bash
curl "http://localhost:8081/api/v1/admin/referral-transactions?status=Pending&limit=20&offset=0" \
  -H "X-Admin-Token: local-admin-token"
```

## Admin: Process Mock Referral Payouts

```bash
curl -X POST http://localhost:8081/api/v1/admin/referrals/process-payouts \
  -H "Content-Type: application/json" \
  -H "X-Admin-Token: local-admin-token" \
  -d '{
    "limit": 20
  }'
```

## WhatsApp Webhook Verification

Set `WHATSAPP_VERIFY_TOKEN` in `.env`, then verify the webhook locally:

```bash
curl "http://localhost:8081/api/v1/whatsapp/webhook?hub.mode=subscribe&hub.verify_token={WHATSAPP_VERIFY_TOKEN}&hub.challenge=local-challenge"
```

Production Meta Cloud API mode uses the same callback URL:

```text
https://api.yourdomain.com/api/v1/whatsapp/webhook
```

Set `WHATSAPP_PROVIDER=meta`, `WHATSAPP_ACCESS_TOKEN`, `WHATSAPP_PHONE_NUMBER_ID`, `WHATSAPP_BUSINESS_ACCOUNT_ID`, and `WHATSAPP_GRAPH_API_VERSION`. See `docs_whatsapp_meta.md` for the full setup, template list, pricing notes, and compliance rules.

Meta webhook message IDs are deduplicated in Redis at `wa_msg:{message_id}`. Status callbacks and unsupported webhook events return `200` with `"ignored": true`.

## WhatsApp/OpenWA Local Message Test

This accepts OpenWA-style JSON in mock mode. No real Meta, OpenWA, or WhatsApp API call is made when `WHATSAPP_PROVIDER=mock`.

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "hi"
  }'
```

To start onboarding with a referral code, send:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543252@c.us",
    "type": "chat",
    "body": "ref RAJU100"
  }'
```

Continue the new-worker onboarding flow by sending replies from the same phone:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "Ravi Kumar"
  }'
```

Language options:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "1"
  }'
```

Supported languages are `English`, `Hindi`, `Marathi`, and `Telugu`.

Continue profile setup:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "Machine Operator"
  }'
```

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "Pune"
  }'
```

## WhatsApp Aadhaar OTP Verification Flow

Choose Aadhaar OTP from the verification menu:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "A"
  }'
```

Send Aadhaar number. Raw Aadhaar is not stored in Redis or PostgreSQL.

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "123456789012"
  }'
```

Give consent and repeat Aadhaar in the same message so the server can start mock OTP without storing raw Aadhaar between messages:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "YES 123456789012"
  }'
```

Verify mock Aadhaar OTP. The local mock accepts any non-empty OTP.

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "123456"
  }'
```

Successful Aadhaar OTP marks the worker as `Low` risk.
If the worker joined with a valid referral code, completing Aadhaar, document upload, or skip verification creates a pending Rs 100 cashback transaction for the referrer.

## WhatsApp Document Upload Verification Flow

Choose document upload:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "B"
  }'
```

Send an OpenWA-style media/document reference. Only `media_ref` is stored.

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "image",
    "media_ref": "local-dev/document-aadhaar-photo.jpg",
    "caption": "Aadhaar document"
  }'
```

Document upload marks the worker as `Medium` risk.

## WhatsApp Skip Verification Flow

Choose skip and confirm:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "C"
  }'
```

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "YES"
  }'
```

Skip verification marks the worker as `High` risk.
If the worker joined with a valid referral code, this also makes the referral cashback eligible.

## WhatsApp Referral Code

Returning users can type `referral` to see their referral code:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "referral"
  }'
```

## WhatsApp Returning User Menu

For a returning worker, any message without an active onboarding state returns:

```text
1. Check Application Status
2. Update Profile
3. Browse Jobs
4. Apply to Job
```

To start application status OTP flow:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "1"
  }'
```

The local mock response includes `Local dev OTP: 123456` style text for testing. Reply with that OTP:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "{otp_from_mock_reply}"
  }'
```

## WhatsApp Browse Jobs And Apply

Browse active jobs from the returning menu:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "3"
  }'
```

Apply by replying with a job ID shown in the browse response:

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "from": "919876543250@c.us",
    "type": "chat",
    "body": "{job_id}"
  }'
```

Application creation uses the existing application service, creates a pending notification event, and sends confirmation through the mock WhatsApp sender.

## WhatsApp Meta-Style Local Message Test

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "entry": [
      {
        "changes": [
          {
            "value": {
              "messages": [
                {
                  "from": "919876543251",
                  "type": "text",
                  "text": {
                    "body": "hi"
                  }
                }
              ]
            }
          }
        ]
      }
    ]
  }'
```
