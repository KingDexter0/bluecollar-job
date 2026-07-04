"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { AuthLayout } from "@/components/AuthLayout";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { apiFetch, setToken } from "@/lib/api";
import type { Employer } from "@/types/api";

export default function LoginPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({ email: "", password: "" });

  async function submit(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const response = await apiFetch<{ employer: Employer; token: string }>("/api/v1/employers/login", {
        method: "POST",
        body: JSON.stringify(form)
      });
      setToken(response.token);
      router.push("/employer/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLayout title="Employer Login" subtitle="Use your JWT-backed employer account.">
      <form className="space-y-4" onSubmit={submit}>
        {error ? <ErrorState message={error} /> : null}
        <Input label="Email" type="email" required value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
        <Input label="Password" type="password" required value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
        <Button className="w-full" disabled={loading}>
          {loading ? "Logging in..." : "Login"}
        </Button>
        <p className="text-center text-sm text-muted">
          Need an account? <Link className="font-semibold text-brand" href="/employer/register">Register</Link>
        </p>
      </form>
    </AuthLayout>
  );
}
