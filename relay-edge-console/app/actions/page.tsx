"use client";

import Link from "next/link";
import { useFleetActions } from "@/hooks/useFleetActions";
import { Badge, Card, Empty, SectionTitle, toneForStatus } from "@/components/UI";

export default function ActionsPage() {
  const { actions, loading } = useFleetActions(200, 10000);

  return (
    <Card>
      <SectionTitle>Action History (all sites)</SectionTitle>
      <div className="flex flex-col">
        {loading && actions.length === 0 && <Empty>Loading…</Empty>}
        {!loading && actions.length === 0 && <Empty>No actions recorded yet.</Empty>}
        {actions.map((a) => (
          <div
            key={`${a.site}-${a.action_id}`}
            className="flex items-center gap-4 py-2 text-sm font-mono border-b border-border/60 last:border-0"
          >
            <span className="text-text-dim w-40 shrink-0">{a.time.replace("T", " ").slice(0, 19)}</span>
            <Link href={`/sites/${encodeURIComponent(a.site)}`} className="w-24 shrink-0 text-cyan hover:underline truncate">
              {a.site}
            </Link>
            <Link
              href={`/sites/${encodeURIComponent(a.site)}/devices/${encodeURIComponent(a.device_id)}`}
              className="w-28 shrink-0 hover:underline truncate"
            >
              {a.device_id}
            </Link>
            <span className="w-32 shrink-0 truncate">{a.target}</span>
            <span className="w-20 shrink-0 text-text-dim">{a.value}</span>
            <span className="w-24 shrink-0 text-text-dim truncate">{a.actor ?? "-"}</span>
            <span className="flex-1" />
            <Badge tone={toneForStatus(a.status)}>{a.status.toUpperCase()}</Badge>
          </div>
        ))}
      </div>
    </Card>
  );
}
