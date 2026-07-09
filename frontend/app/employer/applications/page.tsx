"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { DashboardLayout } from "@/components/DashboardLayout";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Modal } from "@/components/ui/Modal";
import { Select } from "@/components/ui/Select";
import { Table, Td, Th } from "@/components/ui/Table";
import { apiFetch, toRFC3339 } from "@/lib/api";
import { applicationStatuses, verificationTiers } from "@/lib/constants";
import { useEmployerAuth } from "@/lib/useEmployerAuth";
import type { ApplicationStatus, ATSApplication, InterviewSlot, Job, VerificationTier } from "@/types/api";

type Filters = {
  job_id: string;
  status: string;
  verification_tier: string;
  target_role: string;
  preferred_zone: string;
};

const emptyFilters: Filters = { job_id: "", status: "", verification_tier: "", target_role: "", preferred_zone: "" };

export default function EmployerApplicationsPage() {
  const authReady = useEmployerAuth();
  const [applications, setApplications] = useState<ATSApplication[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [filters, setFilters] = useState<Filters>(emptyFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [selected, setSelected] = useState<ATSApplication | null>(null);
  const [directOpen, setDirectOpen] = useState(false);
  const [slotsOpen, setSlotsOpen] = useState(false);
  const [lastSlots, setLastSlots] = useState<InterviewSlot[]>([]);

  const jobOptions = useMemo(() => [{ label: "All jobs", value: "" }, ...jobs.map((job) => ({ label: job.title, value: job.id }))], [jobs]);

  async function loadData(nextFilters = filters) {
    setLoading(true);
    setError("");
    const query = new URLSearchParams();
    query.set("limit", "100");
    Object.entries(nextFilters).forEach(([key, value]) => {
      if (value) {
        query.set(key, value);
      }
    });
    try {
      const [applicationResponse, jobResponse] = await Promise.all([
        apiFetch<{ applications: ATSApplication[] }>(`/api/v1/employer/applications?${query.toString()}`),
        apiFetch<{ jobs: Job[] }>("/api/v1/employer/jobs?limit=100")
      ]);
      setApplications(applicationResponse.applications);
      setJobs(jobResponse.jobs);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load applications");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (authReady) {
      loadData(emptyFilters);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [authReady]);

  async function updateStatus(application: ATSApplication, status: ApplicationStatus) {
    setError("");
    setSuccess("");
    try {
      await apiFetch(`/api/v1/employer/applications/${application.id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status })
      });
      setSuccess("Application status updated");
      await loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update status");
    }
  }

  function applyFilters(event: FormEvent) {
    event.preventDefault();
    loadData(filters);
  }

  return (
    <DashboardLayout title="ATS Applications" subtitle="Review workers, filter applications, and schedule interviews.">
      {!authReady ? <LoadingState label="Checking login..." /> : null}
      <form className="mb-5 grid gap-3 rounded-lg border border-slate-200 bg-white p-4 md:grid-cols-6" onSubmit={applyFilters}>
        <Select label="Job" value={filters.job_id} onChange={(e) => setFilters({ ...filters, job_id: e.target.value })} options={jobOptions} />
        <Select label="Status" value={filters.status} onChange={(e) => setFilters({ ...filters, status: e.target.value })} options={[{ label: "All", value: "" }, ...applicationStatuses.map((status) => ({ label: status, value: status }))]} />
        <Select label="Risk tier" value={filters.verification_tier} onChange={(e) => setFilters({ ...filters, verification_tier: e.target.value })} options={[{ label: "All", value: "" }, ...verificationTiers.map((tier) => ({ label: tier, value: tier }))]} />
        <Input label="Target role" value={filters.target_role} onChange={(e) => setFilters({ ...filters, target_role: e.target.value })} />
        <Input label="Preferred zone" value={filters.preferred_zone} onChange={(e) => setFilters({ ...filters, preferred_zone: e.target.value })} />
        <div className="flex items-end gap-2">
          <Button>Filter</Button>
          <Button type="button" variant="secondary" onClick={() => { setFilters(emptyFilters); loadData(emptyFilters); }}>Reset</Button>
        </div>
      </form>

      {error ? <ErrorState message={error} /> : null}
      {success ? <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-4 text-sm font-medium text-green-700">{success}</div> : null}
      {loading ? <LoadingState /> : null}
      {!loading ? (
        <Table>
          <thead>
            <tr>
              <Th>Worker</Th>
              <Th>Job</Th>
              <Th>Status</Th>
              <Th>Risk</Th>
              <Th>Profile</Th>
              <Th>Actions</Th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {applications.map((application) => (
              <tr key={application.id}>
                <Td>
                  <p className="font-semibold text-ink">{application.worker_full_name}</p>
                  <p className="text-xs text-muted">{application.worker_phone_number}</p>
                </Td>
                <Td>
                  <p>{application.job_title}</p>
                  <p className="text-xs text-muted">{application.job_role}</p>
                </Td>
                <Td><Badge tone="blue">{application.status}</Badge></Td>
                <Td><RiskBadge tier={application.worker_verification_tier} /></Td>
                <Td>
                  <p>{application.worker_target_role || "No target role"}</p>
                  <p className="text-xs text-muted">{application.worker_preferred_zone || "No preferred zone"}</p>
                </Td>
                <Td>
                  <div className="flex flex-wrap gap-2">
                    <select className="rounded-md border border-slate-200 px-2 py-2 text-sm" value={application.status} onChange={(e) => updateStatus(application, e.target.value as ApplicationStatus)}>
                      {applicationStatuses.map((status) => <option key={status} value={status}>{status}</option>)}
                    </select>
                    <Button variant="secondary" onClick={() => { setSelected(application); setDirectOpen(true); }}>Direct</Button>
                    <Button variant="secondary" onClick={() => { setSelected(application); setSlotsOpen(true); }}>3 slots</Button>
                  </div>
                </Td>
              </tr>
            ))}
            {!applications.length ? (
              <tr>
                <Td colSpan={6} className="text-center text-muted">No applications match the current filters.</Td>
              </tr>
            ) : null}
          </tbody>
        </Table>
      ) : null}

      <DirectScheduleModal
        open={directOpen}
        application={selected}
        onClose={() => setDirectOpen(false)}
        onDone={async (slot) => {
          setLastSlots([slot]);
          setSuccess("Interview scheduled");
          setDirectOpen(false);
          await loadData();
        }}
      />
      <SlotScheduleModal
        open={slotsOpen}
        application={selected}
        onClose={() => setSlotsOpen(false)}
        onDone={async (slots) => {
          setLastSlots(slots);
          setSuccess("Three worker-selectable slots created");
          setSlotsOpen(false);
          await loadData();
        }}
      />
      {lastSlots.length ? (
        <Card className="mt-5">
          <CardTitle>Latest Scheduled Slots</CardTitle>
          <div className="mt-4 grid gap-3 md:grid-cols-3">
            {lastSlots.map((slot) => (
              <div key={slot.id} className="rounded-md border border-slate-200 p-3 text-sm">
                <p className="font-semibold">{new Date(slot.starts_at).toLocaleString()} - {new Date(slot.ends_at).toLocaleTimeString()}</p>
                <p className="mt-1 text-muted">{slot.factory_location}</p>
                <Badge tone={slot.status === "Confirmed" ? "green" : slot.status === "Cancelled" ? "red" : "blue"}>{slot.status}</Badge>
              </div>
            ))}
          </div>
        </Card>
      ) : null}
    </DashboardLayout>
  );
}

function RiskBadge({ tier }: { tier: VerificationTier }) {
  const tone = tier === "Low" ? "green" : tier === "Medium" ? "yellow" : "red";
  return <Badge tone={tone}>{tier}</Badge>;
}

function DirectScheduleModal({ open, application, onClose, onDone }: { open: boolean; application: ATSApplication | null; onClose: () => void; onDone: (slot: InterviewSlot) => void }) {
  const [form, setForm] = useState({ date: "", start: "", end: "", factory_location: "", google_maps_url: "" });
  const [error, setError] = useState("");

  useEffect(() => {
    if (open) {
      setForm({ date: "", start: "", end: "", factory_location: "", google_maps_url: "" });
      setError("");
    }
  }, [open, application?.id]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!application) {
      return;
    }
    setError("");
    try {
      const response = await apiFetch<{ interview_slot: InterviewSlot }>(`/api/v1/employer/applications/${application.id}/interview/direct`, {
        method: "POST",
        body: JSON.stringify({
          starts_at: toRFC3339(form.date, form.start),
          ends_at: toRFC3339(form.date, form.end),
          timezone: "Asia/Kolkata",
          factory_location: form.factory_location,
          google_maps_url: form.google_maps_url
        })
      });
      onDone(response.interview_slot);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Scheduling failed");
    }
  }

  return (
    <Modal open={open} title="Direct interview scheduling" onClose={onClose}>
      <form className="space-y-4" onSubmit={submit}>
        {error ? <ErrorState message={error} /> : null}
        <div className="grid gap-3 md:grid-cols-3">
          <Input label="Date" required placeholder="YYYY-MM-DD" value={form.date} onChange={(e) => setForm({ ...form, date: e.target.value })} />
          <Input label="Start time" required placeholder="HH:MM" value={form.start} onChange={(e) => setForm({ ...form, start: e.target.value })} />
          <Input label="End time" required placeholder="HH:MM" value={form.end} onChange={(e) => setForm({ ...form, end: e.target.value })} />
        </div>
        <Input label="Factory location" required value={form.factory_location} onChange={(e) => setForm({ ...form, factory_location: e.target.value })} />
        <Input label="Google Maps URL" required value={form.google_maps_url} onChange={(e) => setForm({ ...form, google_maps_url: e.target.value })} />
        <Button>Schedule interview</Button>
      </form>
    </Modal>
  );
}

function SlotScheduleModal({ open, application, onClose, onDone }: { open: boolean; application: ATSApplication | null; onClose: () => void; onDone: (slots: InterviewSlot[]) => void }) {
  const [date, setDate] = useState("");
  const [location, setLocation] = useState("");
  const [maps, setMaps] = useState("");
  const [times, setTimes] = useState([
    { start: "", end: "" },
    { start: "", end: "" },
    { start: "", end: "" }
  ]);
  const [error, setError] = useState("");

  useEffect(() => {
    if (open) {
      setDate("");
      setLocation("");
      setMaps("");
      setTimes([
        { start: "", end: "" },
        { start: "", end: "" },
        { start: "", end: "" }
      ]);
      setError("");
    }
  }, [open, application?.id]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!application) {
      return;
    }
    setError("");
    try {
      const response = await apiFetch<{ interview_slots: InterviewSlot[] }>(`/api/v1/employer/applications/${application.id}/interview/slots`, {
        method: "POST",
        body: JSON.stringify({
          slots: times.map((time) => ({
            starts_at: toRFC3339(date, time.start),
            ends_at: toRFC3339(date, time.end),
            timezone: "Asia/Kolkata",
            factory_location: location,
            google_maps_url: maps
          }))
        })
      });
      onDone(response.interview_slots);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Slot creation failed");
    }
  }

  return (
    <Modal open={open} title="Create 3 worker-selectable slots" onClose={onClose}>
      <form className="space-y-4" onSubmit={submit}>
        {error ? <ErrorState message={error} /> : null}
        <Input label="Date" required placeholder="YYYY-MM-DD" value={date} onChange={(e) => setDate(e.target.value)} />
        {times.map((time, index) => (
          <div key={index} className="grid gap-3 md:grid-cols-2">
            <Input label={`Slot ${index + 1} start`} required placeholder="HH:MM" value={time.start} onChange={(e) => setTimes(times.map((item, i) => i === index ? { ...item, start: e.target.value } : item))} />
            <Input label={`Slot ${index + 1} end`} required placeholder="HH:MM" value={time.end} onChange={(e) => setTimes(times.map((item, i) => i === index ? { ...item, end: e.target.value } : item))} />
          </div>
        ))}
        <Input label="Factory location" required value={location} onChange={(e) => setLocation(e.target.value)} />
        <Input label="Google Maps URL" required value={maps} onChange={(e) => setMaps(e.target.value)} />
        <Button>Create slots</Button>
      </form>
    </Modal>
  );
}
