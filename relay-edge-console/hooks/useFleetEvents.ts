"use client";

import { useCallback, useEffect, useState } from "react";
import { GatewayClient } from "@/lib/api-client";
import { getGateways } from "@/lib/gateways";
import { RelayEvent } from "@/lib/types";

// Cross-gateway recent-event backlog, polled (not SSE): holding one live
// stream connection open per gateway just for the fleet overview's header
// tiles isn't worth it — the site-detail page is where SSE is used.
export function useFleetEvents(limit = 30, pollMs = 15000) {
  const [events, setEvents] = useState<RelayEvent[]>([]);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(() => {
    const gateways = getGateways();
    Promise.allSettled(gateways.map((gw) => new GatewayClient(gw).events(limit))).then((results) => {
      const merged: RelayEvent[] = [];
      results.forEach((r) => {
        if (r.status === "fulfilled") merged.push(...r.value);
      });
      merged.sort((a, b) => (a.time < b.time ? 1 : -1));
      setEvents(merged.slice(0, limit));
      setLoading(false);
    });
  }, [limit]);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, pollMs);
    return () => clearInterval(id);
  }, [refresh, pollMs]);

  return { events, loading, refresh };
}
