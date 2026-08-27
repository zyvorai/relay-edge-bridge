"use client";

import { useEffect, useRef, useState } from "react";
import { Gateway, RelayEvent } from "@/lib/types";

export type StreamStatus = "connecting" | "live" | "reconnect";

// Hand-rolled fetch-based SSE reader, not the browser's EventSource: this
// endpoint requires an Authorization header, which EventSource cannot set.
// Mirrors the pattern used by the sibling `relay` project's own console
// (frontend/src/api/client.js `openStream`).
export function useGatewayStream(gw: Gateway | undefined, onEvent: (ev: RelayEvent) => void) {
  const [status, setStatus] = useState<StreamStatus>("connecting");
  const onEventRef = useRef(onEvent);

  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    if (!gw) return;
    let cancelled = false;
    let controller: AbortController | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    async function connect() {
      controller = new AbortController();
      setStatus((s) => (s === "live" ? s : "connecting"));
      try {
        const res = await fetch(`${gw!.url}/v1/console/stream`, {
          headers: { Authorization: `Bearer ${gw!.token}` },
          signal: controller.signal,
        });
        if (!res.ok || !res.body) throw new Error(`stream returned ${res.status}`);
        setStatus("live");
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buf = "";
        while (!cancelled) {
          const { done, value } = await reader.read();
          if (done) break;
          buf += decoder.decode(value, { stream: true });
          const frames = buf.split("\n\n");
          buf = frames.pop() ?? "";
          for (const frame of frames) {
            const dataLine = frame.split("\n").find((l) => l.startsWith("data: "));
            if (!dataLine) continue; // keepalive/ready frames carry no event body
            try {
              const ev = JSON.parse(dataLine.slice("data: ".length));
              if (ev && ev.specversion) onEventRef.current(ev as RelayEvent);
            } catch {
              // not a parseable event frame; ignore
            }
          }
        }
      } catch {
        // connection refused/aborted/dropped — fall through to reconnect
      }
      if (!cancelled) {
        setStatus("reconnect");
        retryTimer = setTimeout(connect, 2000);
      }
    }

    connect();
    return () => {
      cancelled = true;
      controller?.abort();
      if (retryTimer) clearTimeout(retryTimer);
    };
  }, [gw]);

  return status;
}
