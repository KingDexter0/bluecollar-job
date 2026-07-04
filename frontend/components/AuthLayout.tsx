import Link from "next/link";

export function AuthLayout({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-surface p-4">
      <div className="w-full max-w-md rounded-lg border border-slate-200 bg-white p-6 shadow-soft">
        <Link href="/" className="text-sm font-semibold text-brand">
          BlueCollarJob
        </Link>
        <h1 className="mt-5 text-2xl font-bold text-ink">{title}</h1>
        <p className="mt-1 text-sm text-muted">{subtitle}</p>
        <div className="mt-6">{children}</div>
      </div>
    </main>
  );
}
