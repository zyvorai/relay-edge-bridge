import { ReactNode } from "react";

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`glass rounded-xl p-5 ${className}`}>{children}</div>;
}

export function SectionTitle({ children }: { children: ReactNode }) {
  return (
    <h2 className="text-xs uppercase tracking-wider text-text-dim mb-3 font-mono">{children}</h2>
  );
}

const TONE_COLOR: Record<string, string> = {
  healthy: "var(--re-healthy)",
  attention: "var(--re-attention)",
  critical: "var(--re-critical)",
  notifying: "var(--re-notifying)",
  escalated: "var(--re-escalated)",
  verified: "var(--re-verified)",
  info: "var(--re-info)",
};

export function toneForStatus(status: string): string {
  switch (status) {
    case "verified":
      return "verified";
    case "executed":
      return "notifying";
    case "failed":
      return "critical";
    case "approval_required":
      return "attention";
    default:
      return "info";
  }
}

export function Dot({ tone }: { tone: string }) {
  const color = TONE_COLOR[tone] ?? TONE_COLOR.info;
  return (
    <span
      className="inline-block w-2 h-2 rounded-full shrink-0"
      style={{ background: color, boxShadow: `0 0 8px ${color}` }}
    />
  );
}

export function Badge({ tone, children }: { tone: string; children: ReactNode }) {
  const color = TONE_COLOR[tone] ?? TONE_COLOR.info;
  return (
    <span
      className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-mono border"
      style={{ color, borderColor: `${color}55`, background: `${color}14` }}
    >
      <Dot tone={tone} />
      {children}
    </span>
  );
}

export function Metric({ label, value, sub }: { label: string; value: ReactNode; sub?: string }) {
  return (
    <div>
      <div className="text-3xl font-display font-semibold tabular-nums">{value}</div>
      <div className="text-xs uppercase tracking-wider text-text-dim mt-1">{label}</div>
      {sub && <div className="text-xs text-text-dim mt-0.5">{sub}</div>}
    </div>
  );
}

// Renders the block-character health meter from the reference mockup
// ("Mumbai ██████████ Healthy"), width cells filled proportionally to pct.
export function HealthBar({ pct, width = 10 }: { pct: number; width?: number }) {
  const filled = Math.round((Math.max(0, Math.min(100, pct)) / 100) * width);
  const tone = pct >= 95 ? "healthy" : pct >= 70 ? "attention" : "critical";
  const color = TONE_COLOR[tone];
  return (
    <span className="font-mono tracking-tight" style={{ color }} title={`${pct.toFixed(1)}%`}>
      {"█".repeat(filled)}
      <span className="text-text-dim opacity-40">{"░".repeat(width - filled)}</span>
    </span>
  );
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="text-sm text-text-dim py-8 text-center font-mono">{children}</div>;
}
