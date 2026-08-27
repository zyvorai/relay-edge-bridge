import { ActionRecord, RelayEvent } from "@/lib/types";
import { Badge, toneForStatus } from "./UI";

function shortTime(iso: string) {
  return iso ? iso.slice(11, 19) : "--:--:--";
}

function deviceFromSubject(subject?: string) {
  return subject?.startsWith("device/") ? subject.slice("device/".length) : (subject ?? "-");
}

function summarizeData(data: unknown): string {
  if (!data || typeof data !== "object") return "";
  const d = data as Record<string, unknown>;
  const values = d.values as Record<string, unknown> | undefined;
  if (values && typeof values === "object") {
    const [k, v] = Object.entries(values)[0] ?? [];
    if (k) return `${k} ${v}`;
  }
  if (typeof d.status === "string") return String(d.status);
  return "";
}

export function EventRow({ ev, site }: { ev: RelayEvent; site?: string }) {
  const isAction = ev.type.startsWith("io.relay.action.");
  return (
    <div className="flex items-center gap-3 py-1.5 text-sm font-mono border-b border-border/60 last:border-0">
      <span className="text-text-dim shrink-0">{shortTime(ev.time)}</span>
      {site && <span className="text-text-dim shrink-0 w-20 truncate">{site}</span>}
      <span className="shrink-0 w-28 truncate">{deviceFromSubject(ev.subject)}</span>
      <span className="text-text-dim shrink-0 w-40 truncate">
        {isAction ? ev.type.replace("io.relay.action.", "action.") : ev.type.replace("io.relay.telemetry.", "")}
      </span>
      <span className="truncate">{summarizeData(ev.data)}</span>
    </div>
  );
}

export function ActionRow({ a, site }: { a: ActionRecord; site?: string }) {
  return (
    <div className="flex items-center gap-3 py-1.5 text-sm font-mono border-b border-border/60 last:border-0">
      <span className="text-text-dim shrink-0">{shortTime(a.time)}</span>
      {site && <span className="text-text-dim shrink-0 w-20 truncate">{site}</span>}
      <span className="shrink-0 w-28 truncate">{a.target}</span>
      <span className="text-text-dim shrink-0 w-28 truncate">{a.device_id}</span>
      <span className="flex-1" />
      <Badge tone={toneForStatus(a.status)}>{a.status.toUpperCase()}</Badge>
    </div>
  );
}
