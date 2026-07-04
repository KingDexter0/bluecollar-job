import type { ApplicationStatus, VerificationTier } from "@/types/api";

export const applicationStatuses: ApplicationStatus[] = [
  "Applied",
  "Shortlisted",
  "Slot_Selection_Pending",
  "Interview_Scheduled",
  "Selected",
  "Rejected"
];

export const verificationTiers: VerificationTier[] = ["Low", "Medium", "High"];

export const languageOptions = [
  { label: "English", value: "en" },
  { label: "Hindi", value: "hi" },
  { label: "Marathi", value: "mr" },
  { label: "Telugu", value: "te" }
];
