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
import { Textarea } from "@/components/ui/Textarea";
import { apiFetch, formatCurrencyPaise } from "@/lib/api";
import { verificationTiers } from "@/lib/constants";
import { useEmployerAuth } from "@/lib/useEmployerAuth";
import type { Job } from "@/types/api";

const emptyJobForm = {
  title: "",
  role: "",
  description: "",
  skill_category: "",
  location_city: "",
  location_state: "",
  shift_schedule: "",
  wage_min_paise: "",
  wage_max_paise: "",
  required_verification_tier: "Low",
  openings: "1",
  is_active: true
};

export default function EmployerJobsPage() {
  const authReady = useEmployerAuth();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [editingJob, setEditingJob] = useState<Job | null>(null);
  const [form, setForm] = useState(emptyJobForm);

  const activeJobs = useMemo(() => jobs.filter((job) => job.is_active).length, [jobs]);

  async function loadJobs() {
    setLoading(true);
    setError("");
    try {
      const response = await apiFetch<{ jobs: Job[] }>("/api/v1/employer/jobs?limit=100");
      setJobs(response.jobs);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load jobs");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (authReady) {
      loadJobs();
    }
  }, [authReady]);

  function openCreate() {
    setEditingJob(null);
    setForm(emptyJobForm);
    setModalOpen(true);
  }

  function openEdit(job: Job) {
    setEditingJob(job);
    setForm({
      title: job.title,
      role: job.role,
      description: job.description,
      skill_category: job.skill_category,
      location_city: job.location_city,
      location_state: job.location_state,
      shift_schedule: job.shift_schedule,
      wage_min_paise: job.wage_min_paise ? String(job.wage_min_paise) : "",
      wage_max_paise: job.wage_max_paise ? String(job.wage_max_paise) : "",
      required_verification_tier: job.required_verification_tier,
      openings: String(job.openings),
      is_active: job.is_active
    });
    setModalOpen(true);
  }

  async function saveJob(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    setSuccess("");
    const payload = {
      ...form,
      openings: Number(form.openings || "1"),
      wage_min_paise: form.wage_min_paise ? Number(form.wage_min_paise) : undefined,
      wage_max_paise: form.wage_max_paise ? Number(form.wage_max_paise) : undefined
    };

    try {
      if (editingJob) {
        await apiFetch(`/api/v1/employer/jobs/${editingJob.id}`, { method: "PATCH", body: JSON.stringify(payload) });
        setSuccess("Job updated");
      } else {
        await apiFetch("/api/v1/employer/jobs", { method: "POST", body: JSON.stringify(payload) });
        setSuccess("Job created");
      }
      setModalOpen(false);
      await loadJobs();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save job");
    } finally {
      setSaving(false);
    }
  }

  async function toggleStatus(job: Job) {
    setError("");
    setSuccess("");
    try {
      await apiFetch(`/api/v1/employer/jobs/${job.id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ is_active: !job.is_active })
      });
      setSuccess(`Job marked ${job.is_active ? "inactive" : "active"}`);
      await loadJobs();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update job status");
    }
  }

  return (
    <DashboardLayout title="Employer Jobs" subtitle="Create, edit, and manage active job postings.">
      {!authReady ? <LoadingState label="Checking login..." /> : null}
      <div className="mb-4 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          {activeJobs >= 7 ? (
            <div className="rounded-md border border-yellow-200 bg-yellow-50 px-3 py-2 text-sm font-medium text-yellow-800">
              Growth tier warning: you have reached 7 active jobs. Enterprise allows unlimited active jobs.
            </div>
          ) : null}
        </div>
        <Button onClick={openCreate}>Create job</Button>
      </div>
      {error ? <ErrorState message={error} /> : null}
      {success ? <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-4 text-sm font-medium text-green-700">{success}</div> : null}
      {loading ? <LoadingState /> : null}
      {!loading ? (
        <Table>
          <thead>
            <tr>
              <Th>Job</Th>
              <Th>Location</Th>
              <Th>Shift</Th>
              <Th>Wage</Th>
              <Th>Status</Th>
              <Th>Actions</Th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {jobs.map((job) => (
              <tr key={job.id}>
                <Td>
                  <p className="font-semibold text-ink">{job.title}</p>
                  <p className="text-xs text-muted">{job.role} · {job.openings} openings</p>
                </Td>
                <Td>{job.location_city}, {job.location_state}</Td>
                <Td>{job.shift_schedule}</Td>
                <Td>{formatCurrencyPaise(job.wage_min_paise)} - {formatCurrencyPaise(job.wage_max_paise)}</Td>
                <Td><Badge tone={job.is_active ? "green" : "gray"}>{job.is_active ? "Active" : "Inactive"}</Badge></Td>
                <Td>
                  <div className="flex flex-wrap gap-2">
                    <Button variant="secondary" onClick={() => openEdit(job)}>Edit</Button>
                    <Button variant={job.is_active ? "danger" : "secondary"} onClick={() => toggleStatus(job)}>
                      {job.is_active ? "Deactivate" : "Activate"}
                    </Button>
                  </div>
                </Td>
              </tr>
            ))}
            {!jobs.length ? (
              <tr>
                <Td colSpan={6} className="text-center text-muted">No jobs found. Create your first job to begin collecting applications.</Td>
              </tr>
            ) : null}
          </tbody>
        </Table>
      ) : null}

      <Modal open={modalOpen} title={editingJob ? "Edit job" : "Create job"} onClose={() => setModalOpen(false)}>
        <form className="space-y-4" onSubmit={saveJob}>
          <div className="grid gap-3 md:grid-cols-2">
            <Input label="Job title" required value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
            <Input label="Role" required value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })} />
          </div>
          <Textarea label="Description" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          <div className="grid gap-3 md:grid-cols-2">
            <Input label="Skill category" value={form.skill_category} onChange={(e) => setForm({ ...form, skill_category: e.target.value })} />
            <Input label="Shift schedule" required value={form.shift_schedule} onChange={(e) => setForm({ ...form, shift_schedule: e.target.value })} />
            <Input label="City" required value={form.location_city} onChange={(e) => setForm({ ...form, location_city: e.target.value })} />
            <Input label="State" required value={form.location_state} onChange={(e) => setForm({ ...form, location_state: e.target.value })} />
            <Input label="Min wage paise" type="number" value={form.wage_min_paise} onChange={(e) => setForm({ ...form, wage_min_paise: e.target.value })} />
            <Input label="Max wage paise" type="number" value={form.wage_max_paise} onChange={(e) => setForm({ ...form, wage_max_paise: e.target.value })} />
            <Input label="Openings" type="number" min="1" value={form.openings} onChange={(e) => setForm({ ...form, openings: e.target.value })} />
            <Select label="Required verification" value={form.required_verification_tier} onChange={(e) => setForm({ ...form, required_verification_tier: e.target.value })} options={verificationTiers.map((tier) => ({ label: tier, value: tier }))} />
          </div>
          <label className="flex items-center gap-2 text-sm font-medium text-slate-700">
            <input type="checkbox" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} />
            Active job
          </label>
          <Button disabled={saving}>{saving ? "Saving..." : "Save job"}</Button>
        </form>
      </Modal>
    </DashboardLayout>
  );
}
