import { Navbar } from "@/components/Navbar";
import { Sidebar } from "@/components/Sidebar";

export function DashboardLayout({ children, title, subtitle }: { children: React.ReactNode; title: string; subtitle?: string }) {
  return (
    <div className="min-h-screen bg-surface">
      <Navbar />
      <div className="flex flex-col md:flex-row">
        <Sidebar />
        <main className="w-full p-4 md:p-6">
          <div className="mb-5">
            <h1 className="text-2xl font-bold text-ink">{title}</h1>
            {subtitle ? <p className="mt-1 text-sm text-muted">{subtitle}</p> : null}
          </div>
          {children}
        </main>
      </div>
    </div>
  );
}
