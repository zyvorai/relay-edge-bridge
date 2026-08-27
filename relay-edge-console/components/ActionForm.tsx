"use client";

import { useState } from "react";
import { GatewayClient } from "@/lib/api-client";
import { DeviceState } from "@/lib/types";

export default function ActionForm({
  client,
  devices,
  defaultDeviceId,
  onDone,
}: {
  client: GatewayClient;
  devices: DeviceState[];
  defaultDeviceId?: string;
  onDone?: () => void;
}) {
  const [deviceId, setDeviceId] = useState(defaultDeviceId ?? devices[0]?.id ?? "");
  const [target, setTarget] = useState("");
  const [value, setValue] = useState("");
  const [actor, setActor] = useState("operator");
  const [pending, setPending] = useState(false);
  const [result, setResult] = useState<string>();

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!deviceId || !target || value === "") return;
    setPending(true);
    setResult(undefined);
    try {
      const res = await client.postAction({
        id: `console-${Date.now()}`,
        device_id: deviceId,
        target,
        value: Number(value),
        actor,
      });
      setResult(`${res.status.toUpperCase()} — verified: ${res.verified}`);
      onDone?.();
    } catch (err) {
      setResult(String(err));
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-2 text-sm">
      <select
        value={deviceId}
        onChange={(e) => setDeviceId(e.target.value)}
        className="bg-transparent border border-border rounded-md px-2 py-1.5 font-mono"
      >
        {devices.map((d) => (
          <option key={d.id} value={d.id} className="bg-panel">
            {d.id}
          </option>
        ))}
      </select>
      <input
        placeholder="target (e.g. rpm_setpoint)"
        value={target}
        onChange={(e) => setTarget(e.target.value)}
        className="bg-transparent border border-border rounded-md px-2 py-1.5 font-mono placeholder:text-text-dim"
      />
      <input
        placeholder="value"
        type="number"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        className="bg-transparent border border-border rounded-md px-2 py-1.5 font-mono placeholder:text-text-dim"
      />
      <input
        placeholder="actor"
        value={actor}
        onChange={(e) => setActor(e.target.value)}
        className="bg-transparent border border-border rounded-md px-2 py-1.5 font-mono placeholder:text-text-dim"
      />
      <button
        type="submit"
        disabled={pending}
        className="mt-1 rounded-md border border-cyan/30 text-cyan bg-cyan/10 px-3 py-1.5 font-mono hover:bg-cyan/20 disabled:opacity-50"
      >
        {pending ? "Sending…" : "Send action"}
      </button>
      {result && <div className="text-xs text-text-dim font-mono mt-1">{result}</div>}
    </form>
  );
}
