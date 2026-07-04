import Link from "next/link";
import { Navbar } from "@/components/Navbar";
import { Card, CardTitle } from "@/components/ui/Card";

const workerBenefits = ["WhatsApp-first onboarding", "Language-friendly profile setup", "Verification tiers that improve trust", "Apply to nearby jobs quickly"];
const employerBenefits = ["Post jobs in minutes", "Track applicants in one ATS", "Schedule interviews without spreadsheets", "Growth and Enterprise subscriptions"];
const steps = ["Employer posts a role", "Worker onboards and applies", "ATS filters by role, zone, and risk", "Interview is scheduled and notifications queue"];

export default function HomePage() {
  return (
    <div className="min-h-screen bg-surface">
      <Navbar />
      <main>
        <section className="bg-gradient-to-br from-teal-900 via-teal-800 to-slate-900 text-white">
          <div className="mx-auto grid max-w-7xl gap-8 px-4 py-16 md:grid-cols-[1.15fr_0.85fr] md:py-24">
            <div>
              <p className="text-sm font-semibold uppercase tracking-wide text-teal-100">Blue-collar hiring, minus the chaos</p>
              <h1 className="mt-4 max-w-3xl text-4xl font-bold leading-tight md:text-6xl">Match verified workers with factory jobs faster.</h1>
              <p className="mt-5 max-w-2xl text-lg text-teal-50">
                A demo-ready platform foundation for worker onboarding, employer job management, ATS, interview scheduling, notifications, and referral cashback.
              </p>
              <div className="mt-8 flex flex-wrap gap-3">
                <Link
                  href="/employer/register"
                  className="inline-flex min-h-10 items-center justify-center rounded-md border border-white bg-white px-4 py-2 text-sm font-semibold text-teal-900 transition hover:bg-teal-50"
                >
                  Employer Register
                </Link>
                <Link
                  href="/employer/login"
                  className="inline-flex min-h-10 items-center justify-center rounded-md border border-white/30 bg-white/10 px-4 py-2 text-sm font-semibold text-white transition hover:bg-white/20"
                >
                  Employer Login
                </Link>
                <Link
                  href="/worker/demo"
                  className="inline-flex min-h-10 items-center justify-center rounded-md px-4 py-2 text-sm font-semibold text-white transition hover:bg-white/10"
                >
                  Worker Demo
                </Link>
              </div>
            </div>
            <div className="rounded-lg border border-white/10 bg-white/10 p-5 backdrop-blur">
              <div className="grid gap-3">
                {["12 min onboarding", "Risk tier visible", "7 active Growth jobs", "Mock WhatsApp ready"].map((item) => (
                  <div key={item} className="rounded-md bg-white/10 p-4 text-sm font-semibold">
                    {item}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section className="mx-auto grid max-w-7xl gap-5 px-4 py-12 md:grid-cols-2">
          <BenefitCard title="For Workers" items={workerBenefits} />
          <BenefitCard title="For Employers" items={employerBenefits} />
        </section>

        <section className="mx-auto max-w-7xl px-4 py-8">
          <h2 className="text-2xl font-bold text-ink">How It Works</h2>
          <div className="mt-5 grid gap-4 md:grid-cols-4">
            {steps.map((step, index) => (
              <Card key={step}>
                <div className="mb-3 flex h-9 w-9 items-center justify-center rounded-md bg-teal-50 font-bold text-brand">{index + 1}</div>
                <p className="font-semibold text-ink">{step}</p>
              </Card>
            ))}
          </div>
        </section>

        <section className="mx-auto max-w-7xl px-4 py-12">
          <h2 className="text-2xl font-bold text-ink">Subscription Pricing</h2>
          <div className="mt-5 grid gap-5 md:grid-cols-2">
            <Card>
              <CardTitle>Growth Tier</CardTitle>
              <p className="mt-3 text-3xl font-bold">₹850/month</p>
              <p className="mt-2 text-sm text-muted">Maximum 7 active job postings. Perfect for small factories and local contractors.</p>
            </Card>
            <Card>
              <CardTitle>Enterprise Tier</CardTitle>
              <p className="mt-3 text-3xl font-bold">₹1,200/month</p>
              <p className="mt-2 text-sm text-muted">Unlimited active jobs for high-volume hiring teams.</p>
            </Card>
          </div>
        </section>
      </main>
    </div>
  );
}

function BenefitCard({ title, items }: { title: string; items: string[] }) {
  return (
    <Card>
      <CardTitle>{title}</CardTitle>
      <ul className="mt-4 space-y-3 text-sm text-slate-700">
        {items.map((item) => (
          <li key={item} className="flex gap-2">
            <span className="mt-1 h-2 w-2 rounded-full bg-brand" />
            <span>{item}</span>
          </li>
        ))}
      </ul>
    </Card>
  );
}
