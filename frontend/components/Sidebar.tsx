import Link from "next/link";

const links = [
  { href: "/employer/dashboard", label: "Dashboard" },
  { href: "/employer/jobs", label: "Jobs" },
  { href: "/employer/applications", label: "Applications" },
  { href: "/worker/demo", label: "Worker Demo" },
  { href: "/admin", label: "Admin Ops" },
  { href: "/dev/notifications", label: "Dev Preview" }
];

const visibleLinks = process.env.NEXT_PUBLIC_APP_ENV === "production" ? links.filter((link) => !link.href.startsWith("/dev")) : links;

export function Sidebar() {
  return (
    <aside className="w-full border-b border-slate-200 bg-white p-3 md:min-h-[calc(100vh-57px)] md:w-64 md:border-b-0 md:border-r">
      <nav className="flex gap-2 overflow-x-auto md:flex-col">
        {visibleLinks.map((link) => (
          <Link key={link.href} href={link.href} className="whitespace-nowrap rounded-md px-3 py-2 text-sm font-medium text-slate-700 hover:bg-teal-50 hover:text-brand">
            {link.label}
          </Link>
        ))}
      </nav>
    </aside>
  );
}
