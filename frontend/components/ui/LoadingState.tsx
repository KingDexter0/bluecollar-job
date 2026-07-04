export function LoadingState({ label = "Loading..." }: { label?: string }) {
  return <div className="rounded-lg border border-slate-200 bg-white p-5 text-sm text-slate-600">{label}</div>;
}
