"use client";

import { FormEvent, useEffect, useState } from "react";
import { Navbar } from "@/components/Navbar";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Table, Td, Th } from "@/components/ui/Table";
import { apiFetch } from "@/lib/api";
import { languageOptions } from "@/lib/constants";
import type { Application, IdentityVerification, Job, Worker } from "@/types/api";

export default function WorkerDemoPage() {
  const [worker, setWorker] = useState<Worker | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [applications, setApplications] = useState<Application[]>([]);
  const [verification, setVerification] = useState<IdentityVerification | null>(null);
  const [aadhaarTransaction, setAadhaarTransaction] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [workerForm, setWorkerForm] = useState({
    phone_number: "+91987654" + Math.floor(1000 + Math.random() * 8999),
    full_name: "Demo Worker",
    language_preference: "en",
    target_role: "Machine Operator",
    preferred_zone: "Pune",
    referred_by_code: ""
  });
  const [aadhaar, setAadhaar] = useState("");
  const [otp, setOtp] = useState("");
  const [documentRef, setDocumentRef] = useState("demo-document-reference");
  const [selectedJobID, setSelectedJobID] = useState("");

  async function loadJobs() {
    const response = await apiFetch<{ jobs: Job[] }>("/api/v1/jobs?limit=50");
    setJobs(response.jobs);
    if (response.jobs[0]) {
      setSelectedJobID(response.jobs[0].id);
    }
  }

  async function loadApplications(workerID: string) {
    const response = await apiFetch<{ applications: Application[] }>(`/api/v1/workers/${workerID}/applications?limit=50`);
    setApplications(response.applications);
  }

  useEffect(() => {
    loadJobs().catch((err) => setError(err instanceof Error ? err.message : "Failed to load jobs"));
  }, []);

  async function createWorker(event: FormEvent) {
    event.preventDefault();
    setError("");
    setMessage("");
    try {
      const payload = {
        ...workerForm,
        target_role: workerForm.target_role,
        preferred_zone: workerForm.preferred_zone,
        referred_by_code: workerForm.referred_by_code || undefined
      };
      const response = await apiFetch<{ worker: Worker }>("/api/v1/workers", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      setWorker(response.worker);
      setMessage(`Worker created with referral code ${response.worker.referral_code}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create worker");
    }
  }

  async function startAadhaar() {
    if (!worker) return;
    setError("");
    setMessage("");
    try {
      const response = await apiFetch<{ identity_verification: IdentityVerification }>(`/api/v1/workers/${worker.id}/identity/aadhaar/start`, {
        method: "POST",
        body: JSON.stringify({ aadhaar_number: aadhaar, consent_given: true })
      });
      setVerification(response.identity_verification);
      setAadhaarTransaction(response.identity_verification.aadhaar_reference_key || "");
      setAadhaar("");
      setMessage("Mock Aadhaar OTP started. Use the transaction/reference shown below.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to start Aadhaar OTP");
    }
  }

  async function verifyAadhaar() {
    if (!worker) return;
    setError("");
    setMessage("");
    try {
      const response = await apiFetch<{ identity_verification: IdentityVerification }>(`/api/v1/workers/${worker.id}/identity/aadhaar/verify`, {
        method: "POST",
        body: JSON.stringify({ transaction_id: aadhaarTransaction, otp })
      });
      setVerification(response.identity_verification);
      const updated = await apiFetch<{ worker: Worker }>(`/api/v1/workers/${worker.id}`);
      setWorker(updated.worker);
      setOtp("");
      setMessage("Aadhaar verified. Worker is now Low risk.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to verify Aadhaar");
    }
  }

  async function documentUpload() {
    if (!worker) return;
    setError("");
    try {
      const response = await apiFetch<{ identity_verification: IdentityVerification }>(`/api/v1/workers/${worker.id}/identity/document`, {
        method: "POST",
        body: JSON.stringify({ document_ref: documentRef })
      });
      setVerification(response.identity_verification);
      const updated = await apiFetch<{ worker: Worker }>(`/api/v1/workers/${worker.id}`);
      setWorker(updated.worker);
      setMessage("Document reference submitted. Worker is now Medium risk.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Document upload failed");
    }
  }

  async function skipVerification() {
    if (!worker) return;
    setError("");
    try {
      const response = await apiFetch<{ identity_verification: IdentityVerification }>(`/api/v1/workers/${worker.id}/identity/skip`, {
        method: "POST",
        body: JSON.stringify({ reason: "Skipped from frontend demo" })
      });
      setVerification(response.identity_verification);
      const updated = await apiFetch<{ worker: Worker }>(`/api/v1/workers/${worker.id}`);
      setWorker(updated.worker);
      setMessage("Verification skipped. Worker is now High risk.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Skip failed");
    }
  }

  async function applyToJob() {
    if (!worker || !selectedJobID) return;
    setError("");
    try {
      await apiFetch<{ application: Application }>("/api/v1/applications", {
        method: "POST",
        body: JSON.stringify({ user_id: worker.id, job_id: selectedJobID, source: "frontend_demo" })
      });
      await loadApplications(worker.id);
      setMessage("Application created. It will appear in employer ATS.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Application failed");
    }
  }

  return (
    <div className="min-h-screen bg-surface">
      <Navbar />
      <main className="mx-auto max-w-7xl px-4 py-6">
        <h1 className="text-2xl font-bold text-ink">Worker Demo</h1>
        <p className="mt-1 text-sm text-muted">Create a worker, verify identity, browse jobs, apply, and check application status.</p>
        {error ? <div className="mt-4"><ErrorState message={error} /></div> : null}
        {message ? <div className="mt-4 rounded-lg border border-green-200 bg-green-50 p-4 text-sm font-medium text-green-700">{message}</div> : null}

        <div className="mt-6 grid gap-5 lg:grid-cols-[0.9fr_1.1fr]">
          <Card>
            <CardTitle>Create worker</CardTitle>
            <form className="mt-4 space-y-4" onSubmit={createWorker}>
              <Input label="Phone number" required value={workerForm.phone_number} onChange={(e) => setWorkerForm({ ...workerForm, phone_number: e.target.value })} />
              <Input label="Full name" required value={workerForm.full_name} onChange={(e) => setWorkerForm({ ...workerForm, full_name: e.target.value })} />
              <Select label="Language" value={workerForm.language_preference} onChange={(e) => setWorkerForm({ ...workerForm, language_preference: e.target.value })} options={languageOptions} />
              <Input label="Target role" value={workerForm.target_role} onChange={(e) => setWorkerForm({ ...workerForm, target_role: e.target.value })} />
              <Input label="Preferred zone" value={workerForm.preferred_zone} onChange={(e) => setWorkerForm({ ...workerForm, preferred_zone: e.target.value })} />
              <Input label="Referred by code" value={workerForm.referred_by_code} onChange={(e) => setWorkerForm({ ...workerForm, referred_by_code: e.target.value })} />
              <Button>Create worker</Button>
            </form>
          </Card>

          <Card>
            <CardTitle>Current worker</CardTitle>
            {worker ? (
              <div className="mt-4 space-y-2 text-sm">
                <p><strong>{worker.full_name}</strong> · {worker.phone_number}</p>
                <p>{worker.target_role} · {worker.preferred_zone}</p>
                <p>Referral code: <strong>{worker.referral_code}</strong></p>
                <p>Risk tier: <RiskBadge tier={worker.verification_tier} /></p>
              </div>
            ) : (
              <p className="mt-4 text-sm text-muted">Create a worker to unlock verification and applications.</p>
            )}
          </Card>
        </div>

        <div className="mt-5 grid gap-5 lg:grid-cols-3">
          <Card>
            <CardTitle>Aadhaar mock OTP</CardTitle>
            <div className="mt-4 space-y-3">
              <Input label="Aadhaar number" placeholder="Enter 12 digits" value={aadhaar} onChange={(e) => setAadhaar(e.target.value)} />
              <Button disabled={!worker} onClick={startAadhaar}>Start OTP</Button>
              <Input label="Transaction ID" value={aadhaarTransaction} onChange={(e) => setAadhaarTransaction(e.target.value)} />
              <Input label="OTP" placeholder="Enter mock OTP" value={otp} onChange={(e) => setOtp(e.target.value)} />
              <Button disabled={!worker || !aadhaarTransaction} onClick={verifyAadhaar}>Verify OTP</Button>
            </div>
          </Card>
          <Card>
            <CardTitle>Document verification</CardTitle>
            <div className="mt-4 space-y-3">
              <Input label="Document reference" value={documentRef} onChange={(e) => setDocumentRef(e.target.value)} />
              <Button disabled={!worker} onClick={documentUpload}>Submit document</Button>
            </div>
          </Card>
          <Card>
            <CardTitle>Skip verification</CardTitle>
            <p className="mt-4 text-sm text-muted">Marks the worker as High risk for demo purposes.</p>
            <Button className="mt-4" disabled={!worker} variant="secondary" onClick={skipVerification}>Skip verification</Button>
          </Card>
        </div>

        {verification ? (
          <Card className="mt-5">
            <CardTitle>Latest verification</CardTitle>
            <pre className="mt-3 overflow-x-auto rounded-md bg-slate-900 p-4 text-xs text-slate-50">{JSON.stringify(verification, null, 2)}</pre>
          </Card>
        ) : null}

        <Card className="mt-5">
          <CardTitle>Browse jobs and apply</CardTitle>
          <div className="mt-4 flex flex-col gap-3 md:flex-row md:items-end">
            <Select label="Job" value={selectedJobID} onChange={(e) => setSelectedJobID(e.target.value)} options={jobs.map((job) => ({ label: `${job.title} · ${job.location_city}`, value: job.id }))} />
            <Button disabled={!worker || !selectedJobID} onClick={applyToJob}>Apply to job</Button>
            <Button variant="secondary" disabled={!worker} onClick={() => worker && loadApplications(worker.id)}>Check application status</Button>
          </div>
          <Table className="mt-4">
            <thead>
              <tr>
                <Th>Job</Th>
                <Th>Role</Th>
                <Th>Location</Th>
                <Th>Shift</Th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {jobs.map((job) => (
                <tr key={job.id}>
                  <Td>{job.title}</Td>
                  <Td>{job.role}</Td>
                  <Td>{job.location_city}, {job.location_state}</Td>
                  <Td>{job.shift_schedule}</Td>
                </tr>
              ))}
              {!jobs.length ? (
                <tr>
                  <Td colSpan={4} className="text-center text-muted">No active jobs available yet.</Td>
                </tr>
              ) : null}
            </tbody>
          </Table>
        </Card>

        <Card className="mt-5">
          <CardTitle>Worker applications</CardTitle>
          <Table className="mt-4">
            <thead>
              <tr>
                <Th>Application</Th>
                <Th>Job ID</Th>
                <Th>Status</Th>
                <Th>Source</Th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {applications.map((application) => (
                <tr key={application.id}>
                  <Td>{application.id}</Td>
                  <Td>{application.job_id}</Td>
                  <Td><Badge tone="blue">{application.status}</Badge></Td>
                  <Td>{application.source}</Td>
                </tr>
              ))}
              {!applications.length ? (
                <tr>
                  <Td colSpan={4} className="text-center text-muted">No applications found for this worker yet.</Td>
                </tr>
              ) : null}
            </tbody>
          </Table>
        </Card>
      </main>
    </div>
  );
}

function RiskBadge({ tier }: { tier: string }) {
  const tone = tier === "Low" ? "green" : tier === "Medium" ? "yellow" : "red";
  return <Badge tone={tone}>{tier}</Badge>;
}
