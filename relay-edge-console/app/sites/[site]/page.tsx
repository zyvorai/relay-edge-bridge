"use client";

import { use, useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { GatewayClient } from "@/lib/api-client";
import { getGateway } from "@/lib/gateways";
import { useGatewayStream } from "@/hooks/useGatewayStream";
import { DeviceState, RelayEvent, SpoolStats, Summary } from "@/lib/types";
import { Badge, Card, Dot, Empty, Metric, SectionTitle } from "@/components/UI";
import { EventRow } from "@/components/Rows";
import ActionForm from "@/components/ActionForm";

export default function SiteDetailPage({ params }: { params: Promise<{ site: string }> }) {
  const { site } = use(params);
  const gw = getGateway(decodeURIComponent(site));
  const client = useMemo(() => (gw ? new GatewayClient(gw) : undefined), [gw]);

  const [summary, setSummary] = useState<Summary>();
  const [devices, setDevices] = useState<DeviceState[]>([]);
  const [spool, setSpool] = useState<SpoolStats>();
  const [events, setEvents] = useState<RelayEvent[]>([]);
  const [error, setError] = useState<string>();

  const refreshRest = useCallback(() => {
    if (!client) return;
    Promise.all([client.summary(), client.devices(), client.spool()])
      .then(([s, d, sp]) => {
        setSummary(s);
        setDevices(d);
        setSpool(sp);
        setError(undefined);
      })
      .catch((e) => setError(String(e)));
  }, [client]);

  const streamStatus = useGatewayStream(gw, (ev) => {
    setEvents((prev) => [ev, ...prev].slice(0, 100));
  });

  // Adaptive polling backstop: fast (3s) while the SSE stream isn't
  // confirmed live, slow (15s) once it is — mirrors relay's own frontend.
  useEffect(() => {
    refreshRest();
    const ms = streamStatus === "live" ? 15000 : 3000;
    const id = setInterval(refreshRest, ms);
    return () => clearInterval(id);
  }, [refreshRest, streamStatus]);

  // Seed the timeline with recent backlog once on load.
  useEffect(() => {
    if (!client) return;
    client.events(50).then(setEvents).catch(() => {});
  }, [client]);

  if (!gw) {
    return <Empty>Unknown site &quot;{site}&quot;. Check config/gateways.json.</Empty>;
  }

  return (
    <div className="flex flex-col gap-6">
      <Card>
        <div className="flex items-center justify-between flex-wrap gap-4">
          <div className="flex gap-10">
            <Metric label="Site" value={summary?.site ?? gw.name} />
            <Metric label="Devices" value={summary ? `${summary.healthy_count}/${summary.device_count}` : "-"} />
            <Metric label="Spool depth" value={spool?.depth ?? "-"} />
          </div>
          <Badge tone={streamStatus === "live" ? "healthy" : "attention"}>
            {streamStatus === "live" ? "LIVE STREAM" : streamStatus.toUpperCase()}
          </Badge>
        </div>
        {error && <div className="text-xs text-critical mt-3 font-mono">{error}</div>}
      </Card>

      <Card>
        <SectionTitle>Devices</SectionTitle>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {devices.map((d) => (
            <Link
              key={d.id}
              href={`/sites/${encodeURIComponent(gw.name)}/devices/${encodeURIComponent(d.id)}`}
              className="rounded-lg border border-border p-3 hover:border-border-strong transition-colors block"
            >
              <div className="flex items-center gap-2">
                <Dot tone={d.connected ? "healthy" : "critical"} />
                <span className="font-mono text-sm">{d.id}</span>
              </div>
              <div className="text-xs text-text-dim mt-1">{d.protocol}</div>
              {d.last_values && (
                <div className="text-xs font-mono mt-2 text-text-dim truncate">
                  {Object.entries(d.last_values)
                    .slice(0, 3)
                    .map(([k, v]) => `${k}=${v}`)
                    .join("  ")}
                </div>
              )}
              {!d.connected && d.last_error && (
                <div className="text-xs text-critical mt-1 truncate">{d.last_error}</div>
              )}
            </Link>
          ))}
          {devices.length === 0 && <Empty>No devices reported yet.</Empty>}
        </div>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <SectionTitle>Event Timeline</SectionTitle>
          <div className="flex flex-col max-h-96 overflow-y-auto">
            {events.length === 0 ? <Empty>No events yet.</Empty> : events.map((ev) => <EventRow key={ev.id} ev={ev} />)}
          </div>
        </Card>
        <div className="flex flex-col gap-6">
          <Card>
            <SectionTitle>Send Action</SectionTitle>
            {client && <ActionForm client={client} devices={devices} onDone={refreshRest} />}
          </Card>
          <Card>
            <SectionTitle>Offline / Store-and-Forward</SectionTitle>
            {spool ? (
              <div className="font-mono text-sm flex flex-col gap-1">
                <div>depth: {spool.depth}</div>
                {spool.oldest_pending_age_seconds !== undefined && (
                  <div>oldest pending: {spool.oldest_pending_age_seconds.toFixed(1)}s ago</div>
                )}
                {spool.priorities && (
                  <div>
                    by priority:{" "}
                    {Object.entries(spool.priorities)
                      .map(([p, n]) => `p${p}=${n}`)
                      .join(" ")}
                  </div>
                )}
              </div>
            ) : (
              <Empty>No spool data.</Empty>
            )}
          </Card>
        </div>
      </div>
    </div>
  );
}
