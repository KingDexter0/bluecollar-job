import type { HTMLAttributes } from "react";

type BadgeProps = HTMLAttributes<HTMLSpanElement> & {
  tone?: "green" | "yellow" | "red" | "blue" | "gray";
};

export function Badge({ tone = "gray", className = "", ...props }: BadgeProps) {
  const tones = {
    green: "bg-green-50 text-green-700 ring-green-600/20",
    yellow: "bg-yellow-50 text-yellow-800 ring-yellow-600/20",
    red: "bg-red-50 text-red-700 ring-red-600/20",
    blue: "bg-blue-50 text-blue-700 ring-blue-600/20",
    gray: "bg-slate-100 text-slate-700 ring-slate-600/20"
  };
  return <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ring-1 ${tones[tone]} ${className}`} {...props} />;
}
