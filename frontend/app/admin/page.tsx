"use client";

import { FormEvent, useEffect, useState } from "react";
import { DashboardLayout } from "@/components/DashboardLayout";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Select } from "@/components/ui/Select";
import { Table, Td, Th } from "@/components/ui/Table";
import { apiFetch, clearAdminToken, getAdminToken, setAdminToken } from "@/lib/api";

type AdminSummary = {
  total_workers: number;
  total_employers: number;
  total_jobs: number;
  total_applications: number;
  total_referrals: number;
  total_notification_events: number;
  pending_notifications: number;
  failed_notifications: number;
  cashback_pending: number;
  cashback_paid: number;
  cashback_failed: number;
  interviews_scheduled: number;
  applications_by_status: Record<string, number>;
  workers_by_verification_tier: Record<string, number>;
  jobs_by_active_status: Record<string, number>;
  referrals_by_payout_status: Record<string, number>;
};

type NotificationRow = {
  id: string;
  worker_id?: string;
  phone_number?: string;
  event_type: string;
  message_preview: string;
  status: "Pending" | "Processing" | "Sent" | "Failed";
  failure_reason?: string;
  created_at: string;
  processed_at?: string;
};

type ReferralTransaction = {
  id: string;
  referral_id: string;
  user_id: string;
  amount_paise: number;
  currency: string;
  status: "Pending" | "Processing" | "Paid" | "Failed";
  external_reference?: string;
  created_at: string;
  paid_at?: string;
};

type ProcessResult = {
  claimed: number;
  paid: number;
  failed: number;
};

const notificationStatuses = ["Pending", "Processing", "Sent", "Failed"];
const referralStatuses = ["Pending", "Processing", "Paid", "Failed"];

export default function AdminPage() {
  const [token, setTokenValue] = useState("local-admin-token");
  const [summary, setSummary] = useState<AdminSummary | null>(null);
  const [notifications, setNotifications] = useState<NotificationRow[]>([]);
  const [referrals, setReferrals] = useState<ReferralTransaction[]>([]);
  const [notificationFilters, setNotificationFilters] = useState({ status: "", event_type: "" });
  const [referralStatus, setReferralStatus] = useState("");
  const [payoutResult, setPayoutResult] = useState<ProcessResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [processing, setProcessing] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const savedToken = getAdminToken();
    if (savedToken) {
      setTokenValue(savedToken);
      loadAdminData();
    } else {
      setAdminToken("local-admin-token");
      loadAdminData();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function loadAdminData() {
    setLoading(true);
    setError("");
    try {
      const [summaryResponse, notificationResponse, referralResponse] = await Promise.all([
        apiFetch<{ summary: AdminSummary }>("/api/v1/admin/summary"),
        apiFetch<{ notifications: NotificationRow[] }>(buildNotificationPath(notificationFilters)),
        apiFetch<{ referral_transactions: ReferralTransaction[] }>(buildReferralPath(referralStatus))
      ]);
      setSummary(summaryResponse.summary);
      setNotifications(notificationResponse.notifications);
      setReferrals(referralResponse.referral_transactions);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load admin data");
    } finally {
      setLoading(false);
    }
  }

  async function saveToken(event: FormEvent) {
    event.preventDefault();
    const clean = token.trim();
    if (!clean) {
      clearAdminToken();
      setError("Admin token is required.");
      return;
    }
    setAdminToken(clean);
    await loadAdminData();
  }

  async function applyNotificationFilters(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const response = await apiFetch<{ notifications: NotificationRow[] }>(buildNotificationPath(notificationFilters));
      setNotifications(response.notifications);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load notifications");
    } finally {
      setLoading(false);
    }
  }

  async function applyReferralFilters(nextStatus = referralStatus) {
    setReferralStatus(nextStatus);
    setLoading(true);
    setError("");
    try {
      const response = await apiFetch<{ referral_transactions: ReferralTransaction[] }>(buildReferralPath(nextStatus));
      setReferrals(response.referral_transactions);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load referral transactions");
    } finally {
      setLoading(false);
    }
  }

  async function processPayouts() {
    setProcessing(true);
    setError("");
    try {
      const response = await apiFetch<{ result: ProcessResult }>("/api/v1/admin/referrals/process-payouts", {
        method: "POST",
        body: JSON.stringify({ limit: 20 })
      });
      setPayoutResult(response.result);
      await loadAdminData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to process payouts");
    } finally {
      setProcessing(false);
    }
  }

  return (
    <DashboardLayout title="Admin Operations" subtitle="Demo operations view for platform health, notifications, referrals, and funnel analytics.">
      {error ? <ErrorState message={error} /> : null}

      <Card>
        <CardTitle>Admin access</CardTitle>
        <form className="mt-4 grid gap-3 md:grid-cols-[1fr_auto_auto]" onSubmit={saveToken}>
          <Input
            label="Admin token"
            value={token}
            onChange={(event) => setTokenValue(event.target.value)}
            placeholder="local-admin-token"
          />
          <div className="flex items-end">
            <Button>Save token</Button>
          </div>
          <div className="flex items-end">
            <Button type="button" variant="secondary" onClick={loadAdminData}>Refresh</Button>
          </div>
        </form>
        <p className="mt-2 text-sm text-muted">Local demo token: <strong>local-admin-token</strong>. Use a long random `ADMIN_TOKEN` outside local development.</p>
      </Card>

      {loading && !summary ? <div className="mt-5"><LoadingState label="Loading admin overview..." /></div> : null}

      {summary ? (
        <>
          <div className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <Metric label="Workers" value={summary.total_workers} />
            <Metric label="Employers" value={summary.total_employers} />
            <Metric label="Jobs" value={summary.total_jobs} />
            <Metric label="Applications" value={summary.total_applications} />
            <Metric label="Referrals" value={summary.total_referrals} />
            <Metric label="Notifications" value={summary.total_notification_events} />
            <Metric label="Pending Notifications" value={summary.pending_notifications} tone="yellow" />
            <Metric label="Failed Notifications" value={summary.failed_notifications} tone="red" />
            <Metric label="Cashback Pending" value={summary.cashback_pending} tone="yellow" />
            <Metric label="Cashback Paid" value={summary.cashback_paid} tone="green" />
            <Metric label="Cashback Failed" value={summary.cashback_failed} tone="red" />
            <Metric label="Interviews Scheduled" value={summary.interviews_scheduled} tone="blue" />
          </div>

          <div className="mt-5 grid gap-5 xl:grid-cols-2">
            <AnalyticsCard title="Applications by status" data={summary.applications_by_status} />
            <AnalyticsCard title="Workers by verification tier" data={summary.workers_by_verification_tier} />
            <AnalyticsCard title="Jobs active/inactive" data={summary.jobs_by_active_status} />
            <AnalyticsCard title="Referral payouts" data={summary.referrals_by_payout_status} />
          </div>
        </>
      ) : null}

      <div className="mt-5 grid gap-5 xl:grid-cols-2">
        <Card>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <CardTitle>Notifications</CardTitle>
            <Button type="button" variant="secondary" onClick={loadAdminData}>Reload</Button>
          </div>
          <form className="mt-4 grid gap-3 md:grid-cols-[170px_1fr_auto]" onSubmit={applyNotificationFilters}>
            <Select
              label="Status"
              value={notificationFilters.status}
              onChange={(event) => setNotificationFilters({ ...notificationFilters, status: event.target.value })}
              options={[{ label: "All statuses", value: "" }, ...notificationStatuses.map((status) => ({ label: status, value: status }))]}
            />
            <Input
              label="Event type"
              value={notificationFilters.event_type}
              onChange={(event) => setNotificationFilters({ ...notificationFilters, event_type: event.target.value })}
              placeholder="interview_scheduled"
            />
            <div className="flex items-end">
              <Button>Filter</Button>
            </div>
          </form>
          <Table className="mt-4">
            <thead>
              <tr>
                <Th>Event</Th>
                <Th>Recipient</Th>
                <Th>Status</Th>
                <Th>Preview</Th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {notifications.map((notification) => (
                <tr key={notification.id}>
                  <Td>
                    <p className="font-semibold text-ink">{notification.event_type}</p>
                    <p className="text-xs text-muted">{formatDate(notification.created_at)}</p>
                  </Td>
                  <Td>{notification.phone_number || "-"}</Td>
                  <Td><NotificationBadge status={notification.status} /></Td>
                  <Td className="max-w-xs">{notification.failure_reason || notification.message_preview}</Td>
                </tr>
              ))}
              {!notifications.length ? (
                <tr><Td colSpan={4} className="text-center text-muted">No notification events found.</Td></tr>
              ) : null}
            </tbody>
          </Table>
        </Card>

        <Card>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <CardTitle>Referral cashback</CardTitle>
            <Button type="button" onClick={processPayouts} disabled={processing}>{processing ? "Processing..." : "Process payouts"}</Button>
          </div>
          {payoutResult ? (
            <div className="mt-3 grid grid-cols-3 gap-3">
              <MiniMetric label="Claimed" value={payoutResult.claimed} />
              <MiniMetric label="Paid" value={payoutResult.paid} />
              <MiniMetric label="Failed" value={payoutResult.failed} />
            </div>
          ) : null}
          <div className="mt-4 max-w-xs">
            <Select
              label="Payout status"
              value={referralStatus}
              onChange={(event) => applyReferralFilters(event.target.value)}
              options={[{ label: "All statuses", value: "" }, ...referralStatuses.map((status) => ({ label: status, value: status }))]}
            />
          </div>
          <Table className="mt-4">
            <thead>
              <tr>
                <Th>Transaction</Th>
                <Th>Amount</Th>
                <Th>Status</Th>
                <Th>Created</Th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {referrals.map((transaction) => (
                <tr key={transaction.id}>
                  <Td>
                    <p className="font-semibold text-ink">{transaction.id}</p>
                    <p className="text-xs text-muted">Worker {transaction.user_id}</p>
                  </Td>
                  <Td>{formatINR(transaction.amount_paise)}</Td>
                  <Td><ReferralBadge status={transaction.status} /></Td>
                  <Td>{formatDate(transaction.created_at)}</Td>
                </tr>
              ))}
              {!referrals.length ? (
                <tr><Td colSpan={4} className="text-center text-muted">No referral cashback transactions found.</Td></tr>
              ) : null}
            </tbody>
          </Table>
        </Card>
      </div>
    </DashboardLayout>
  );
}

function buildNotificationPath(filters: { status: string; event_type: string }) {
  const query = new URLSearchParams({ limit: "50" });
  if (filters.status) {
    query.set("status", filters.status);
  }
  if (filters.event_type.trim()) {
    query.set("event_type", filters.event_type.trim());
  }
  return `/api/v1/admin/notifications?${query.toString()}`;
}

function buildReferralPath(status: string) {
  const query = new URLSearchParams({ limit: "50" });
  if (status) {
    query.set("status", status);
  }
  return `/api/v1/admin/referral-transactions?${query.toString()}`;
}

function Metric({ label, value, tone = "gray" }: { label: string; value: number; tone?: "green" | "yellow" | "red" | "blue" | "gray" }) {
  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <p className="text-sm text-muted">{label}</p>
        <Badge tone={tone}>{label.includes("Failed") ? "watch" : "live"}</Badge>
      </div>
      <p className="mt-3 text-3xl font-bold text-ink">{value}</p>
    </Card>
  );
}

function MiniMetric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-slate-200 p-3">
      <p className="text-xs text-muted">{label}</p>
      <p className="mt-1 text-xl font-bold text-ink">{value}</p>
    </div>
  );
}

function AnalyticsCard({ title, data }: { title: string; data: Record<string, number> }) {
  const entries = Object.entries(data || {});
  const max = Math.max(1, ...entries.map(([, value]) => value));
  return (
    <Card>
      <CardTitle>{title}</CardTitle>
      <div className="mt-4 space-y-3">
        {entries.map(([label, value]) => (
          <div key={label}>
            <div className="mb-1 flex justify-between text-sm">
              <span className="font-medium text-ink">{label}</span>
              <span className="text-muted">{value}</span>
            </div>
            <div className="h-2 rounded-full bg-slate-100">
              <div className="h-2 rounded-full bg-brand" style={{ width: `${Math.max(6, (value / max) * 100)}%` }} />
            </div>
          </div>
        ))}
        {!entries.length ? <p className="text-sm text-muted">No data available yet.</p> : null}
      </div>
    </Card>
  );
}

function NotificationBadge({ status }: { status: NotificationRow["status"] }) {
  const tone = status === "Sent" ? "green" : status === "Failed" ? "red" : status === "Processing" ? "blue" : "yellow";
  return <Badge tone={tone}>{status}</Badge>;
}

function ReferralBadge({ status }: { status: ReferralTransaction["status"] }) {
  const tone = status === "Paid" ? "green" : status === "Failed" ? "red" : status === "Processing" ? "blue" : "yellow";
  return <Badge tone={tone}>{status}</Badge>;
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}

function formatINR(value: number) {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 0
  }).format(value / 100);
}
