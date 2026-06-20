DROP TABLE IF EXISTS notification_events;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS referral_transactions;
DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS interview_slots;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS employers;
DROP TABLE IF EXISTS worker_identity_verifications;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS set_updated_at();

DROP TYPE IF EXISTS notification_status_enum;
DROP TYPE IF EXISTS interview_slot_status_enum;
DROP TYPE IF EXISTS subscription_tier_enum;
DROP TYPE IF EXISTS application_status_enum;
DROP TYPE IF EXISTS identity_verification_status_enum;
DROP TYPE IF EXISTS identity_verification_method_enum;
DROP TYPE IF EXISTS verification_tier_enum;
