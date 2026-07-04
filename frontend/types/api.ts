export type VerificationTier = "Low" | "Medium" | "High";
export type ApplicationStatus =
  | "Applied"
  | "Shortlisted"
  | "Slot_Selection_Pending"
  | "Interview_Scheduled"
  | "Selected"
  | "Rejected";
export type InterviewSlotStatus = "Available" | "Locked" | "Confirmed" | "Cancelled";

export type Employer = {
  id: string;
  company_name: string;
  contact_name: string;
  email: string;
  phone_number?: string;
  city?: string;
  state?: string;
  is_verified: boolean;
};

export type Worker = {
  id: string;
  phone_number: string;
  full_name: string;
  language_preference: string;
  target_role?: string;
  preferred_zone?: string;
  verification_tier: VerificationTier;
  referral_code: string;
  is_active: boolean;
};

export type Job = {
  id: string;
  employer_id: string;
  title: string;
  role: string;
  description: string;
  skill_category: string;
  location_city: string;
  location_state: string;
  shift_schedule: string;
  wage_min_paise?: number;
  wage_max_paise?: number;
  required_verification_tier: VerificationTier;
  openings: number;
  is_active: boolean;
};

export type Application = {
  id: string;
  user_id: string;
  job_id: string;
  employer_id: string;
  status: ApplicationStatus;
  source: string;
  applied_at: string;
};

export type ATSApplication = Application & {
  worker_full_name: string;
  worker_phone_number: string;
  worker_verification_tier: VerificationTier;
  worker_target_role?: string;
  worker_preferred_zone?: string;
  job_title: string;
  job_role: string;
};

export type InterviewSlot = {
  id: string;
  application_id: string;
  starts_at: string;
  ends_at: string;
  timezone: string;
  factory_location?: string;
  google_maps_url?: string;
  status: InterviewSlotStatus;
};

export type IdentityVerification = {
  id: string;
  user_id: string;
  method: string;
  status: string;
  aadhaar_last4?: string;
  aadhaar_masked?: string;
  aadhaar_reference_key?: string;
  document_uploaded: boolean;
  consent_given: boolean;
};

export type APIError = {
  error?: {
    code: string;
    message: string;
  };
};
