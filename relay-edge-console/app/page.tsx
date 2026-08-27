"use client";

import Link from "next/link";
import { useFleetSummary } from "@/hooks/useFleetSummary";
import { useFleetEvents } from "@/hooks/useFleetEvents";
import { useFleetActions } from "@/hooks/useFleetActions";
import { Badge, Card, Empty, HealthBar, Metric, SectionTitle } from "@/components/UI";
import { EventRow, ActionRow } from "@/components/Rows";

export default function FleetOverviewPage() {
  const { sites, totalDevices, healthyPct, degradedSites, loading } = useFleetSummary();
  const { events } = useFleetEvents(30);
  const { actions } = useFleetActions(20);

  const failedAction = actions.find((a) => a.status === "failed");

  return (
    <div className="flex flex-col gap-6">
      <Card>
        <div className="flex flex-wrap gap-10">
          <Metric label="Sites" value={loading ? "-" : sites.length} />
          <Metric label="Devices" value={loading ? "-" : totalDevices} />
          <Metric
            label="Healthy"
            value={loading ? "-" : `${healthyPct.toFixed(1)}%`}
            sub={degradedSites > 0 ? `${degradedSites} site(s) unreachable` : undefined}
          />
        </div>
      </Card>

      {failedAction && (
        <Card className="border-critical/40">
          <div className="flex items-center justify-between gap-4 flex-wrap">
            <div>
              <Badge tone="critical">FAILED ACTION</Badge>
              <div className="mt-2 font-mono text-sm">
                {failedAction.target} on {failedAction.device_id} ({failedAction.site})
              </div>
              {failedAction.detail && (
                <div className="text-xs text-text-dim mt-1">{failedAction.detail}</div>
              )}
            </div>
            <Link
              href={`/sites/${encodeURIComponent(failedAction.site)}/devices/${encodeURIComponent(failedAction.device_id)}`}
              className="text-sm text-cyan hover:underline shrink-0"
            >
              Investigate &rarr;
            </Link>
          </div>
        </Card>
      )}

      <Card>
        <SectionTitle>Fleet Health</SectionTitle>
        <div className="flex flex-col gap-2">
          {sites.map((s) => (
            <Link
              key={s.name}
              href={`/sites/${encodeURIComponent(s.name)}`}
              className="flex items-center gap-4 py-1.5 hover:bg-white/5 rounded-md px-2 -mx-2"
            >
              <span className="w-32 font-mono text-sm truncate">{s.name}</span>
              {s.status === "online" && s.summary ? (
                <>
                  <HealthBar
                    pct={
                      s.summary.device_count > 0
                        ? (s.summary.healthy_count / s.summary.device_count) * 100
                        : 100
                    }
                  />
                  <span className="text-xs text-text-dim ml-2">
                    {s.summary.healthy_count}/{s.summary.device_count} devices
                  </span>
                </>
              ) : (
                <Badge tone="critical">UNREACHABLE</Badge>
              )}
            </Link>
          ))}
          {sites.length === 0 && !loading && <Empty>No gateways configured.</Empty>}
        </div>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <SectionTitle>Live Events</SectionTitle>
          <div className="flex flex-col">
            {events.length === 0 ? (
              <Empty>No events yet.</Empty>
            ) : (
              events.map((ev) => <EventRow key={ev.id} ev={ev} site={ev.site} />)
            )}
          </div>
        </Card>
        <Card>
          <SectionTitle>Actions</SectionTitle>
          <div className="flex flex-col">
            {actions.length === 0 ? (
              <Empty>No actions yet.</Empty>
            ) : (
              actions.map((a) => <ActionRow key={`${a.site}-${a.action_id}`} a={a} site={a.site} />)
            )}
          </div>
        </Card>
      </div>
    </div>
  );
}
