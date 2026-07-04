"use client";

import { useEffect, useMemo, useState } from "react";
import { DashboardLayout } from "@/components/DashboardLayout";
import { Card } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { apiFetch } from "@/lib/api";
import { useEmployerAuth } from "@/lib/useEmployerAuth";
import type { ATSApplication, Job } from "@/types/api";

export default function EmployerDashboardPage() {
  const authReady = useEmployerAuth();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [applications, setApplications] = useState<ATSApplication[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!authReady) {
      return;
    }
    async function load() {
      try {
        const [jobResponse, applicationResponse] = await Promise.all([
          apiFetch<{ jobs: Job[] }>("/api/v1/employer/jobs?limit=100"),
          apiFetch<{ applications: ATSApplication[] }>("/api/v1/employer/applications?limit=100")
        ]);
        setJobs(jobResponse.jobs);
        setApplications(applicationResponse.applications);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load dashboard");
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [authReady]);

  const summary = useMemo(
    () => [
      { label: "Total jobs", value: jobs.length },
      { label: "Active jobs", value: jobs.filter((job) => job.is_active).length },
      { label: "Total applications", value: applications.length },
      { label: "Shortlisted", value: applications.filter((app) => app.status === "Shortlisted").length },
      { label: "Interview scheduled", value: applications.filter((app) => app.status === "Interview_Scheduled").length },
      { label: "Selected", value: applications.filter((app) => app.status === "Selected").length },
      { label: "Rejected", value: applications.filter((app) => app.status === "Rejected").length }
    ],
    [jobs, applications]
  );

  return (
    <DashboardLayout title="Employer Dashboard" subtitle="Hiring activity from your employer account.">
      {!authReady ? <LoadingState label="Checking login..." /> : null}
      {loading ? <LoadingState /> : null}
      {error ? <ErrorState message={error} /> : null}
      {!loading && !error ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {summary.map((item) => (
            <Card key={item.label}>
              <p className="text-sm font-medium text-muted">{item.label}</p>
              <p className="mt-3 text-3xl font-bold text-ink">{item.value}</p>
            </Card>
          ))}
        </div>
      ) : null}
    </DashboardLayout>
  );
}
