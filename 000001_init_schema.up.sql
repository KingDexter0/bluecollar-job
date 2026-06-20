CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TYPE verification_tier_enum AS ENUM ('Low', 'Medium', 'High');

CREATE TYPE identity_verification_method_enum AS ENUM (
    'Aadhaar_OTP',
    'Document_Upload',
    'Skipped'
);

CREATE TYPE identity_verification_status_enum AS ENUM (
    'Pending',
    'OTP_Sent',
    'Verified',
    'Failed',
    'Document_Uploaded',
    'Skipped'
);

CREATE TYPE application_status_enum AS ENUM (
    'Applied',
    'Shortlisted',
    'Slot_Selection_Pending',
    'Interview_Scheduled',
    'Selected',
    'Rejected'
);

CREATE TYPE subscription_tier_enum AS ENUM ('Growth', 'Enterprise');

CREATE TYPE interview_slot_status_enum AS ENUM (
    'Available',
    'Locked',
    'Confirmed',
    'Cancelled'
);

CREATE TYPE notification_status_enum AS ENUM (
    'Pending',
    'Processing',
    'Sent',
    'Failed'
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(20) NOT NULL,
    full_name VARCHAR(150) NOT NULL,
    language_preference VARCHAR(20) NOT NULL DEFAULT 'en',
    target_role VARCHAR(120),
    preferred_zone VARCHAR(120),
    verification_tier verification_tier_enum NOT NULL DEFAULT 'High',
    referral_code VARCHAR(32) NOT NULL,
    referred_by_code VARCHAR(32),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_phone_number_unique UNIQUE (phone_number),
    CONSTRAINT users_referral_code_unique UNIQUE (referral_code),
    CONSTRAINT users_referred_by_code_fk FOREIGN KEY (referred_by_code)
        REFERENCES users(referral_code) ON DELETE SET NULL,
    CONSTRAINT users_phone_number_check CHECK (phone_number ~ '^\+[1-9][0-9]{7,14}$'),
    CONSTRAINT users_referral_self_check CHECK (
        referred_by_code IS NULL OR referred_by_code <> referral_code
    )
);

CREATE TABLE worker_identity_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method identity_verification_method_enum NOT NULL,
    status identity_verification_status_enum NOT NULL DEFAULT 'Pending',
    aadhaar_last4 CHAR(4),
    aadhaar_masked VARCHAR(20),
    aadhaar_hash CHAR(64),
    aadhaar_reference_key VARCHAR(120),
    document_ref TEXT,
    consent_given BOOLEAN NOT NULL DEFAULT FALSE,
    consent_given_at TIMESTAMPTZ,
    verified_at TIMESTAMPTZ,
    failed_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT worker_identity_aadhaar_last4_check CHECK (
        aadhaar_last4 IS NULL OR aadhaar_last4 ~ '^[0-9]{4}$'
    ),
    CONSTRAINT worker_identity_aadhaar_hash_check CHECK (
        aadhaar_hash IS NULL OR aadhaar_hash ~ '^[a-f0-9]{64}$'
    ),
    CONSTRAINT worker_identity_consent_timestamp_check CHECK (
        (consent_given = FALSE AND consent_given_at IS NULL)
        OR (consent_given = TRUE AND consent_given_at IS NOT NULL)
    ),
    CONSTRAINT worker_identity_verified_timestamp_check CHECK (
        verified_at IS NULL OR status = 'Verified'
    ),
    CONSTRAINT worker_identity_document_ref_check CHECK (
        method <> 'Document_Upload'
        OR document_ref IS NOT NULL
    ),
    CONSTRAINT worker_identity_otp_data_check CHECK (
        method <> 'Aadhaar_OTP'
        OR (
            consent_given = TRUE
            AND aadhaar_last4 IS NOT NULL
            AND aadhaar_masked IS NOT NULL
            AND aadhaar_hash IS NOT NULL
            AND aadhaar_reference_key IS NOT NULL
        )
    ),
    CONSTRAINT worker_identity_skipped_check CHECK (
        method <> 'Skipped'
        OR (
            status = 'Skipped'
            AND aadhaar_last4 IS NULL
            AND aadhaar_masked IS NULL
            AND aadhaar_hash IS NULL
            AND aadhaar_reference_key IS NULL
            AND document_ref IS NULL
        )
    )
);

CREATE UNIQUE INDEX idx_worker_identity_one_active_per_user
ON worker_identity_verifications(user_id)
WHERE status IN ('Pending', 'OTP_Sent', 'Document_Uploaded');

CREATE TABLE employers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name VARCHAR(180) NOT NULL,
    contact_name VARCHAR(150) NOT NULL,
    email CITEXT NOT NULL,
    phone_number VARCHAR(20),
    city VARCHAR(100),
    state VARCHAR(100),
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT employers_email_unique UNIQUE (email),
    CONSTRAINT employers_phone_number_check CHECK (
        phone_number IS NULL OR phone_number ~ '^\+[1-9][0-9]{7,14}$'
    )
);

CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employer_id UUID NOT NULL REFERENCES employers(id) ON DELETE CASCADE,
    title VARCHAR(180) NOT NULL,
    description TEXT NOT NULL,
    skill_category VARCHAR(120) NOT NULL,
    location_city VARCHAR(100) NOT NULL,
    location_state VARCHAR(100) NOT NULL,
    wage_min_paise INTEGER CHECK (wage_min_paise IS NULL OR wage_min_paise >= 0),
    wage_max_paise INTEGER CHECK (wage_max_paise IS NULL OR wage_max_paise >= 0),
    required_verification_tier verification_tier_enum NOT NULL DEFAULT 'Low',
    openings INTEGER NOT NULL DEFAULT 1 CHECK (openings > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    published_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT jobs_id_employer_id_unique UNIQUE (id, employer_id),
    CONSTRAINT jobs_wage_range_check CHECK (
        wage_min_paise IS NULL
        OR wage_max_paise IS NULL
        OR wage_min_paise <= wage_max_paise
    ),
    CONSTRAINT jobs_expiry_check CHECK (
        expires_at IS NULL OR published_at IS NULL OR published_at < expires_at
    )
);

CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    employer_id UUID NOT NULL REFERENCES employers(id) ON DELETE CASCADE,
    status application_status_enum NOT NULL DEFAULT 'Applied',
    source VARCHAR(50) NOT NULL DEFAULT 'platform',
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT applications_user_job_unique UNIQUE (user_id, job_id),
    CONSTRAINT applications_job_employer_fk FOREIGN KEY (job_id, employer_id)
        REFERENCES jobs(id, employer_id) ON DELETE CASCADE
);

CREATE TABLE interview_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    timezone VARCHAR(80) NOT NULL DEFAULT 'Asia/Kolkata',
    status interview_slot_status_enum NOT NULL DEFAULT 'Available',
    locked_until TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT interview_slots_time_check CHECK (starts_at < ends_at),
    CONSTRAINT interview_slots_lock_check CHECK (
        locked_until IS NULL OR status IN ('Locked', 'Confirmed')
    ),
    CONSTRAINT interview_slots_confirmed_check CHECK (
        confirmed_at IS NULL OR status = 'Confirmed'
    )
);

CREATE TABLE referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    referred_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    referral_code VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    converted_at TIMESTAMPTZ,
    CONSTRAINT referrals_code_fk FOREIGN KEY (referral_code)
        REFERENCES users(referral_code) ON DELETE RESTRICT,
    CONSTRAINT referrals_distinct_users_check CHECK (
        referred_user_id IS NULL OR referrer_user_id <> referred_user_id
    )
);

CREATE TABLE referral_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID NOT NULL REFERENCES referrals(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_paise INTEGER NOT NULL DEFAULT 10000 CHECK (amount_paise > 0),
    currency CHAR(3) NOT NULL DEFAULT 'INR',
    status VARCHAR(30) NOT NULL DEFAULT 'Pending',
    external_reference VARCHAR(120),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    CONSTRAINT referral_transactions_status_check CHECK (
        status IN ('Pending', 'Processing', 'Paid', 'Failed', 'Cancelled')
    )
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employer_id UUID NOT NULL REFERENCES employers(id) ON DELETE CASCADE,
    tier subscription_tier_enum NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ,
    job_post_limit INTEGER CHECK (job_post_limit IS NULL OR job_post_limit > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT subscriptions_time_check CHECK (ends_at IS NULL OR starts_at < ends_at)
);

CREATE TABLE notification_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    employer_id UUID REFERENCES employers(id) ON DELETE SET NULL,
    application_id UUID REFERENCES applications(id) ON DELETE SET NULL,
    channel VARCHAR(30) NOT NULL,
    event_type VARCHAR(80) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status notification_status_enum NOT NULL DEFAULT 'Pending',
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT notification_events_channel_check CHECK (
        channel IN ('whatsapp', 'sms', 'email', 'system')
    )
);

CREATE TRIGGER trg_users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_worker_identity_verifications_set_updated_at
BEFORE UPDATE ON worker_identity_verifications
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_employers_set_updated_at
BEFORE UPDATE ON employers
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_jobs_set_updated_at
BEFORE UPDATE ON jobs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_applications_set_updated_at
BEFORE UPDATE ON applications
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_interview_slots_set_updated_at
BEFORE UPDATE ON interview_slots
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_subscriptions_set_updated_at
BEFORE UPDATE ON subscriptions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_notification_events_set_updated_at
BEFORE UPDATE ON notification_events
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_users_phone_number ON users(phone_number);
CREATE INDEX idx_users_referral_code ON users(referral_code);
CREATE INDEX idx_users_verification_tier ON users(verification_tier);
CREATE INDEX idx_worker_identity_verifications_user_id ON worker_identity_verifications(user_id);
CREATE INDEX idx_worker_identity_verifications_status ON worker_identity_verifications(status);
CREATE INDEX idx_worker_identity_verifications_method ON worker_identity_verifications(method);
CREATE INDEX idx_employers_email ON employers(email);
CREATE INDEX idx_jobs_employer_id ON jobs(employer_id);
CREATE INDEX idx_jobs_is_active ON jobs(is_active);
CREATE INDEX idx_applications_user_id ON applications(user_id);
CREATE INDEX idx_applications_job_id ON applications(job_id);
CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_interview_slots_application_id ON interview_slots(application_id);
CREATE INDEX idx_interview_slots_starts_at ON interview_slots(starts_at);
CREATE INDEX idx_referrals_referrer_user_id ON referrals(referrer_user_id);
CREATE INDEX idx_referrals_referred_user_id ON referrals(referred_user_id);
CREATE INDEX idx_referral_transactions_referral_id ON referral_transactions(referral_id);
CREATE INDEX idx_referral_transactions_user_id ON referral_transactions(user_id);
CREATE INDEX idx_subscriptions_employer_id ON subscriptions(employer_id);
CREATE INDEX idx_notification_events_status ON notification_events(status);
CREATE INDEX idx_notification_events_scheduled_at ON notification_events(scheduled_at);

INSERT INTO users (
    id,
    phone_number,
    full_name,
    language_preference,
    target_role,
    preferred_zone,
    verification_tier,
    referral_code,
    referred_by_code
) VALUES
    (
        '11111111-1111-1111-1111-111111111111',
        '+919876543210',
        'Raju Kumar',
        'hi',
        'Electrician Helper',
        'Gurugram',
        'Low',
        'RAJU100',
        NULL
    ),
    (
        '22222222-2222-2222-2222-222222222222',
        '+919876543211',
        'Sita Devi',
        'hi',
        'Housekeeping Staff',
        'Noida',
        'Medium',
        'SITA100',
        'RAJU100'
    ),
    (
        '33333333-3333-3333-3333-333333333333',
        '+919876543212',
        'Imran Sheikh',
        'en',
        'Warehouse Helper',
        'Delhi NCR',
        'High',
        'IMRAN100',
        NULL
    );

INSERT INTO worker_identity_verifications (
    id,
    user_id,
    method,
    status,
    aadhaar_last4,
    aadhaar_masked,
    aadhaar_hash,
    aadhaar_reference_key,
    document_ref,
    consent_given,
    consent_given_at,
    verified_at,
    failed_reason
) VALUES
    (
        '44444444-4444-4444-4444-444444444444',
        '11111111-1111-1111-1111-111111111111',
        'Aadhaar_OTP',
        'Verified',
        '1234',
        'XXXX-XXXX-1234',
        '7a51d064967d1ef28095b24e0d6c8f03f2c1f3d318d371d03a9fdc8b4d6c2a91',
        'mock-ekyc-ref-raju-001',
        NULL,
        TRUE,
        NOW() - INTERVAL '1 day',
        NOW() - INTERVAL '1 day',
        NULL
    ),
    (
        '55555555-5555-5555-5555-555555555555',
        '22222222-2222-2222-2222-222222222222',
        'Document_Upload',
        'Document_Uploaded',
        '5678',
        'XXXX-XXXX-5678',
        '8b9f4f7a0d4dd86843a3f6c5d63f04dddf7a7cd72a7c88c70b2f9611f4b85f64',
        NULL,
        's3://bluecollarjob-local-documents/mock/sita-aadhaar-upload.jpg',
        TRUE,
        NOW() - INTERVAL '12 hours',
        NULL,
        'Aadhaar linked mobile unavailable; document uploaded for manual review.'
    ),
    (
        '66666666-6666-6666-6666-666666666666',
        '33333333-3333-3333-3333-333333333333',
        'Skipped',
        'Skipped',
        NULL,
        NULL,
        NULL,
        NULL,
        NULL,
        FALSE,
        NULL,
        NULL,
        'Worker skipped identity verification during onboarding.'
    );

INSERT INTO employers (
    id,
    company_name,
    contact_name,
    email,
    phone_number,
    city,
    state,
    is_verified
) VALUES (
    '77777777-7777-7777-7777-777777777777',
    'Reliable Facilities Pvt Ltd',
    'Anita Sharma',
    'hiring@reliablefacilities.example',
    '+911234567890',
    'Gurugram',
    'Haryana',
    TRUE
);

INSERT INTO jobs (
    id,
    employer_id,
    title,
    description,
    skill_category,
    location_city,
    location_state,
    wage_min_paise,
    wage_max_paise,
    required_verification_tier,
    openings,
    is_active,
    published_at
) VALUES
    (
        '88888888-8888-8888-8888-888888888888',
        '77777777-7777-7777-7777-777777777777',
        'Electrician Helper',
        'Assist senior electricians with wiring, fixtures, and maintenance at commercial sites.',
        'Electrical',
        'Gurugram',
        'Haryana',
        1800000,
        2400000,
        'Low',
        3,
        TRUE,
        NOW()
    ),
    (
        '99999999-9999-9999-9999-999999999999',
        '77777777-7777-7777-7777-777777777777',
        'Housekeeping Staff',
        'Maintain cleanliness for office premises with rotational shifts.',
        'Housekeeping',
        'Noida',
        'Uttar Pradesh',
        1400000,
        1800000,
        'Medium',
        5,
        TRUE,
        NOW()
    );

INSERT INTO applications (
    id,
    user_id,
    job_id,
    employer_id,
    status,
    source
) VALUES
    (
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        '11111111-1111-1111-1111-111111111111',
        '88888888-8888-8888-8888-888888888888',
        '77777777-7777-7777-7777-777777777777',
        'Shortlisted',
        'seed'
    ),
    (
        'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
        '22222222-2222-2222-2222-222222222222',
        '99999999-9999-9999-9999-999999999999',
        '77777777-7777-7777-7777-777777777777',
        'Applied',
        'seed'
    );

INSERT INTO interview_slots (
    id,
    application_id,
    starts_at,
    ends_at,
    status
) VALUES (
    'cccccccc-cccc-cccc-cccc-cccccccccccc',
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    NOW() + INTERVAL '2 days',
    NOW() + INTERVAL '2 days 30 minutes',
    'Available'
);

INSERT INTO referrals (
    id,
    referrer_user_id,
    referred_user_id,
    referral_code,
    converted_at
) VALUES (
    'dddddddd-dddd-dddd-dddd-dddddddddddd',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    'RAJU100',
    NOW()
);

INSERT INTO referral_transactions (
    id,
    referral_id,
    user_id,
    amount_paise,
    status
) VALUES (
    'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee',
    'dddddddd-dddd-dddd-dddd-dddddddddddd',
    '11111111-1111-1111-1111-111111111111',
    10000,
    'Pending'
);

INSERT INTO subscriptions (
    id,
    employer_id,
    tier,
    starts_at,
    job_post_limit,
    is_active
) VALUES (
    'ffffffff-ffff-ffff-ffff-ffffffffffff',
    '77777777-7777-7777-7777-777777777777',
    'Growth',
    NOW(),
    25,
    TRUE
);

INSERT INTO notification_events (
    id,
    user_id,
    application_id,
    channel,
    event_type,
    recipient,
    payload,
    status
) VALUES (
    '12121212-1212-1212-1212-121212121212',
    '11111111-1111-1111-1111-111111111111',
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    'whatsapp',
    'application_shortlisted',
    '+919876543210',
    '{"template": "application_shortlisted", "job_title": "Electrician Helper"}'::jsonb,
    'Pending'
);
