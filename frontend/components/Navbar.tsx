"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/Button";
import { clearToken, getToken } from "@/lib/api";

const showDevLinks = process.env.NEXT_PUBLIC_APP_ENV !== "production";

export function Navbar() {
	const router = useRouter();
	const [hasToken, setHasToken] = useState(false);

	useEffect(() => {
		setHasToken(Boolean(getToken()));
	}, []);

	return (
    <header className="sticky top-0 z-40 border-b border-slate-200 bg-white/95 backdrop-blur">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
        <Link href="/" className="text-lg font-bold text-brand">
          BlueCollarJob
        </Link>
        <nav className="hidden items-center gap-5 text-sm font-medium text-slate-600 md:flex">
          <Link href="/employer/jobs">Jobs</Link>
          <Link href="/employer/applications">ATS</Link>
          <Link href="/worker/demo">Worker Demo</Link>
          <Link href="/admin">Admin</Link>
          {showDevLinks ? <Link href="/dev/notifications">Dev</Link> : null}
        </nav>
        <div className="flex items-center gap-2">
          {hasToken ? (
            <Button
              variant="secondary"
              onClick={() => {
                clearToken();
                router.push("/employer/login");
              }}
            >
              Logout
            </Button>
          ) : (
            <Link href="/employer/login" className="rounded-md border border-slate-200 px-3 py-2 text-sm font-semibold">
              Login
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
