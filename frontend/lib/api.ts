import type { APIError } from "@/types/api";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8081";
const TOKEN_KEY = "bluecollar_employer_token";
const ADMIN_TOKEN_KEY = "bluecollar_admin_token";

export function getToken() {
  if (typeof window === "undefined") {
    return "";
  }
  return window.localStorage.getItem(TOKEN_KEY) || "";
}

export function setToken(token: string) {
  window.localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  window.localStorage.removeItem(TOKEN_KEY);
}

export function getAdminToken() {
  if (typeof window === "undefined") {
    return "";
  }
  return window.localStorage.getItem(ADMIN_TOKEN_KEY) || "";
}

export function setAdminToken(token: string) {
  window.localStorage.setItem(ADMIN_TOKEN_KEY, token);
}

export function clearAdminToken() {
  window.localStorage.removeItem(ADMIN_TOKEN_KEY);
}

export async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const adminToken = getAdminToken();
  const headers = new Headers(options.headers);
  const isProtectedEmployerRoute = path.startsWith("/api/v1/employer/") || path.startsWith("/api/v1/employers/me");
  const isAdminRoute = path.startsWith("/api/v1/admin/");
  if (!headers.has("Content-Type") && options.body) {
    headers.set("Content-Type", "application/json");
  }
  if (token && isProtectedEmployerRoute) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  if (adminToken && isAdminRoute) {
    headers.set("X-Admin-Token", adminToken);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers
  });

  const text = await response.text();
  const body = text ? JSON.parse(text) : {};
  if (!response.ok) {
    const apiError = body as APIError;
    throw new Error(apiError.error?.message || `Request failed with ${response.status}`);
  }
  return body as T;
}

export function formatCurrencyPaise(value?: number) {
  if (!value) {
    return "Not set";
  }
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    maximumFractionDigits: 0
  }).format(value / 100);
}

export function toRFC3339(date: string, time: string) {
  const local = new Date(`${date}T${time}:00`);
  return local.toISOString();
}
