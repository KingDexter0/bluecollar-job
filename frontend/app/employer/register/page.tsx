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

export default function RegisterPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    company_name: "",
    contact_name: "",
    email: "",
    password: "",
    phone_number: "",
    city: "",
    state: ""
  });

  async function submit(event: FormEvent) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const response = await apiFetch<{ employer: Employer; token: string }>("/api/v1/employers/register", {
        method: "POST",
        body: JSON.stringify(form)
      });
      setToken(response.token);
      router.push("/employer/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthLayout title="Register Employer" subtitle="Create a local demo employer account.">
      <form className="space-y-4" onSubmit={submit}>
        {error ? <ErrorState message={error} /> : null}
        <Input label="Company name" required value={form.company_name} onChange={(e) => setForm({ ...form, company_name: e.target.value })} />
        <Input label="Contact name" value={form.contact_name} onChange={(e) => setForm({ ...form, contact_name: e.target.value })} />
        <Input label="Email" type="email" required value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
        <Input label="Password" type="password" required value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
        <Input label="Phone number" placeholder="+919876543210" value={form.phone_number} onChange={(e) => setForm({ ...form, phone_number: e.target.value })} />
        <div className="grid gap-3 md:grid-cols-2">
          <Input label="City" value={form.city} onChange={(e) => setForm({ ...form, city: e.target.value })} />
          <Input label="State" value={form.state} onChange={(e) => setForm({ ...form, state: e.target.value })} />
        </div>
        <Button className="w-full" disabled={loading}>
          {loading ? "Creating..." : "Register"}
        </Button>
        <p className="text-center text-sm text-muted">
          Already registered? <Link className="font-semibold text-brand" href="/employer/login">Login</Link>
        </p>
      </form>
    </AuthLayout>
  );
}
