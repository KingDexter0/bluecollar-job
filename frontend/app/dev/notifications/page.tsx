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
import { apiFetch } from "@/lib/api";

type ProcessResult = {
  claimed: number;
  sent: number;
  failed: number;
};

type NotificationStatus = "Pending" | "Processing" | "Sent" | "Failed";

type NotificationRow = {
  id: string;
  user_id?: string;
  worker_id?: string;
  phone_number?: string;
  event_type: string;
  message_preview: string;
  status: NotificationStatus;
  failure_reason?: string;
  created_at: string;
  updated_at: string;
  processed_at?: string;
};

const statusOptions = ["Pending", "Processing", "Sent", "Failed"];

export default function DevNotificationsPage() {
  const [result, setResult] = useState<ProcessResult | null>(null);
  const [notifications, setNotifications] = useState<NotificationRow[]>([]);
  const [filters, setFilters] = useState({ status: "", event_type: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [tableLoading, setTableLoading] = useState(true);

  async function loadNotifications(nextFilters = filters) {
    setTableLoading(true);
    setError("");
    const query = new URLSearchParams({ limit: "50" });
    if (nextFilters.status) {
      query.set("status", nextFilters.status);
    }
    if (nextFilters.event_type.trim()) {
      query.set("event_type", nextFilters.event_type.trim());
    }
    try {
      const response = await apiFetch<{ notifications: NotificationRow[] }>(`/api/v1/dev/notifications?${query.toString()}`);
      setNotifications(response.notifications);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load notifications");
    } finally {
      setTableLoading(false);
    }
  }

  useEffect(() => {
    loadNotifications();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function processOnce() {
    setLoading(true);
    setError("");
    try {
      const response = await apiFetch<{ result: ProcessResult }>("/api/v1/dev/notifications/process-once", {
        method: "POST",
        body: JSON.stringify({ limit: 10 })
      });
      setResult(response.result);
      await loadNotifications();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to process notifications");
    } finally {
      setLoading(false);
    }
  }

  function applyFilters(event: FormEvent) {
    event.preventDefault();
    loadNotifications(filters);
  }

  if (process.env.NEXT_PUBLIC_APP_ENV === "production") {
    return (
      <DashboardLayout title="Dev Preview Disabled" subtitle="Development-only tools are hidden in production builds.">
        <ErrorState message="Dev notification tools are disabled in production." />
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout title="Notification Dev Preview" subtitle="Local-only notification worker controls.">
      {error ? <ErrorState message={error} /> : null}
      <Card>
        <CardTitle>Process notification events once</CardTitle>
        <p className="mt-2 text-sm text-muted">
          Process pending notification events with the mock WhatsApp sender, then inspect the safe notification preview table below.
        </p>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <Button onClick={processOnce} disabled={loading}>{loading ? "Processing..." : "Process once"}</Button>
          <Badge tone="gray">Pending</Badge>
          <Badge tone="blue">Processing</Badge>
          <Badge tone="green">Sent</Badge>
          <Badge tone="red">Failed</Badge>
        </div>
        {result ? (
          <div className="mt-5 grid gap-3 sm:grid-cols-3">
            <Metric label="Claimed" value={result.claimed} />
            <Metric label="Sent" value={result.sent} />
            <Metric label="Failed" value={result.failed} />
          </div>
        ) : null}
      </Card>

      <Card className="mt-5">
        <CardTitle>Notification events</CardTitle>
        <form className="mt-4 grid gap-3 md:grid-cols-[220px_1fr_auto_auto]" onSubmit={applyFilters}>
          <Select
            label="Status"
            value={filters.status}
            onChange={(event) => setFilters({ ...filters, status: event.target.value })}
            options={[{ label: "All statuses", value: "" }, ...statusOptions.map((status) => ({ label: status, value: status }))]}
          />
          <Input
            label="Event type"
            placeholder="application_submitted"
            value={filters.event_type}
            onChange={(event) => setFilters({ ...filters, event_type: event.target.value })}
          />
          <div className="flex items-end">
            <Button>Filter</Button>
          </div>
          <div className="flex items-end">
            <Button type="button" variant="secondary" onClick={() => { const cleared = { status: "", event_type: "" }; setFilters(cleared); loadNotifications(cleared); }}>
              Reset
            </Button>
          </div>
        </form>

        {tableLoading ? <div className="mt-4"><LoadingState label="Loading notifications..." /></div> : null}
        {!tableLoading ? (
          <Table className="mt-4">
            <thead>
              <tr>
                <Th>Event</Th>
                <Th>Recipient</Th>
                <Th>Status</Th>
                <Th>Message Preview</Th>
                <Th>Failure</Th>
                <Th>Created</Th>
                <Th>Processed</Th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {notifications.map((notification) => (
                <tr key={notification.id}>
                  <Td>
                    <p className="font-semibold text-ink">{notification.event_type}</p>
                    <p className="text-xs text-muted">{notification.id}</p>
                  </Td>
                  <Td>
                    <p>{notification.phone_number || "Not set"}</p>
                    <p className="text-xs text-muted">{notification.worker_id ? `Worker ${notification.worker_id}` : "No worker"}</p>
                  </Td>
                  <Td><NotificationStatusBadge status={notification.status} /></Td>
                  <Td className="max-w-sm">{notification.message_preview}</Td>
                  <Td>{notification.failure_reason || "-"}</Td>
                  <Td>{formatDate(notification.created_at)}</Td>
                  <Td>{notification.processed_at ? formatDate(notification.processed_at) : "-"}</Td>
                </tr>
              ))}
              {!notifications.length ? (
                <tr>
                  <Td colSpan={7} className="text-center text-muted">No notification events match the current filters.</Td>
                </tr>
              ) : null}
            </tbody>
          </Table>
        ) : null}
      </Card>
    </DashboardLayout>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border border-slate-200 p-4">
      <p className="text-sm text-muted">{label}</p>
      <p className="mt-2 text-2xl font-bold text-ink">{value}</p>
    </div>
  );
}

function NotificationStatusBadge({ status }: { status: NotificationStatus }) {
  const tone = status === "Pending" ? "gray" : status === "Processing" ? "blue" : status === "Sent" ? "green" : "red";
  return <Badge tone={tone}>{status}</Badge>;
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}
