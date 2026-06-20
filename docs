# Architecture

This backend follows Clean Architecture boundaries:

- `cmd/api`: application entrypoint and HTTP server wiring
- `internal/config`: environment-based configuration
- `internal/database`: PostgreSQL connection setup
- `internal/cache`: Redis connection setup
- `internal/models`: domain and persistence models
- `internal/repository`: data access implementations
- `internal/service`: business logic and use cases
- `internal/handler`: HTTP handlers
- `internal/middleware`: HTTP middleware
- `migrations`: PostgreSQL schema migrations

The current phase intentionally exposes only the health endpoint. WhatsApp onboarding, dashboards, ATS workflows, and payments will be added behind these same boundaries.

## Database Schema

The initial PostgreSQL migration creates the Day 2 relational foundation:

- `users`: worker records with unique `phone_number`, unique `referral_code`, target role, preferred zone, and default High verification risk
- `worker_identity_verifications`: Aadhaar OTP, document upload, or skipped identity verification metadata without raw Aadhaar storage
- `employers`: employer records with unique `email`
- `jobs`: employer-owned jobs with wage range, openings, active flag, and required verification tier
- `applications`: links users to jobs and employers with application status tracking
- `interview_slots`: interview windows with availability/lock/confirmation state
- `referrals`: worker referral relationships using the referrer's referral code
- `referral_transactions`: cashback ledger records for referral payouts
- `subscriptions`: employer subscription tier state
- `notification_events`: durable notification queue records for future messaging integrations

Seed data is included for local development: three workers covering Low, Medium, and High risk verification outcomes, one employer, two jobs, two applications, and related scheduler/referral/subscription/notification records.

## Aadhaar OTP Redis Design

- `aadhaar_otp:{phone_number}` stores only a hashed OTP or authorized gateway transaction id
- TTL is 5 minutes
- retry state should use `aadhaar_otp_retry:{phone_number}`
- rate limiting should use `rate_limit:aadhaar_otp:{phone_number}`
- raw Aadhaar numbers and OTPs must never be logged or stored permanently
