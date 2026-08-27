"use client";

import { use, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { GatewayClient } from "@/lib/api-client";
import { getGateway } from "@/lib/gateways";
import { ActionRecord, ConfigDevice, DeviceState } from "@/lib/types";
import { Badge, Card, Dot, Empty, SectionTitle, toneForStatus } from "@/components/UI";
import ActionForm from "@/components/ActionForm";

export default function DeviceDetailPage({
  params,
}: {
  params: Promise<{ site: string; id: string }>;
}) {
  const { site, id } = use(params);
  const gw = getGateway(decodeURIComponent(site));
  const deviceId = decodeURIComponent(id);
  const client = useMemo(() => (gw ? new GatewayClient(gw) : undefined), [gw]);

  const [device, setDevice] = useState<DeviceState>();
  const [recentActions, setRecentActions] = useState<ActionRecord[]>([]);
  const [config, setConfig] = useState<ConfigDevice>();

  useEffect(() => {
    if (!client) return;
    let cancelled = false;
    function refresh() {
      client!.device(deviceId).then((r) => {
        if (cancelled) return;
        setDevice(r.device);
        setRecentActions(r.recent_actions ?? []);
      });
    }
    refresh();
    client.configDevices().then((devs) => {
      if (!cancelled) setConfig(devs.find((d) => d.id === deviceId));
    });
    const t = setInterval(refresh, 3000);
    return () => {
      cancelled = true;
      clearInterval(t);
    };
  }, [client, deviceId]);

  if (!gw) return <Empty>Unknown site &quot;{site}&quot;.</Empty>;

  return (
    <div className="flex flex-col gap-6">
      <div className="text-sm">
        <Link href={`/sites/${encodeURIComponent(gw.name)}`} className="text-cyan hover:underline">
          &larr; {gw.name}
        </Link>
      </div>

      <Card>
        <div className="flex items-center gap-3">
          <Dot tone={device?.connected ? "healthy" : "critical"} />
          <span className="font-mono text-lg">{deviceId}</span>
          {config && <span className="text-xs text-text-dim">{config.protocol} &middot; {config.endpoint}</span>}
        </div>
        {device?.last_error && <div className="text-xs text-critical mt-2 font-mono">{device.last_error}</div>}
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <SectionTitle>Digital Shadow</SectionTitle>
          {device?.last_values ? (
            <div className="flex flex-col gap-1 font-mono text-sm">
              {Object.entries(device.last_values).map(([k, v]) => (
                <div key={k} className="flex justify-between border-b border-border/60 py-1">
                  <span className="text-text-dim">{k}</span>
                  <span>{String(v)}</span>
                </div>
              ))}
            </div>
          ) : (
            <Empty>No values reported yet.</Empty>
          )}
          {device?.last_seen && (
            <div className="text-xs text-text-dim mt-3">last seen {device.last_seen}</div>
          )}
        </Card>

        <Card>
          <SectionTitle>Protocol Config</SectionTitle>
          {config ? (
            <div className="font-mono text-sm flex flex-col gap-1">
              <div>protocol: {config.protocol}</div>
              <div>endpoint: {config.endpoint}</div>
              {config.topic && <div>topic: {config.topic}</div>}
              {config.signals && config.signals.length > 0 && (
                <div className="mt-2">
                  <div className="text-text-dim mb-1">signals</div>
                  {config.signals.map((s) => (
                    <div key={s.name} className="flex justify-between">
                      <span>{s.name}</span>
                      <span className="text-text-dim">{s.writable ? "writable" : "read-only"}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <Empty>No config found.</Empty>
          )}
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <SectionTitle>Recent Actions</SectionTitle>
          <div className="flex flex-col gap-2">
            {recentActions.length === 0 ? (
              <Empty>No actions for this device yet.</Empty>
            ) : (
              recentActions.map((a) => (
                <div key={a.action_id} className="flex items-center justify-between font-mono text-sm border-b border-border/60 py-1.5 last:border-0">
                  <span>{a.target} &rarr; {a.value}</span>
                  <Badge tone={toneForStatus(a.status)}>{a.status.toUpperCase()}</Badge>
                </div>
              ))
            )}
          </div>
        </Card>
        <Card>
          <SectionTitle>Send Action</SectionTitle>
          {client && device && <ActionForm client={client} devices={[device]} defaultDeviceId={device.id} />}
        </Card>
      </div>
    </div>
  );
}
