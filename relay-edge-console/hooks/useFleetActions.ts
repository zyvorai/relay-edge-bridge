"use client";

import { useCallback, useEffect, useState } from "react";
import { GatewayClient } from "@/lib/api-client";
import { getGateways } from "@/lib/gateways";
import { ActionRecord } from "@/lib/types";

export type FleetAction = ActionRecord & { site: string };

// Cross-gateway action history: fan out to every configured gateway's
// /v1/console/actions and merge, newest first. Used by both the fleet
// overview's recent-actions panel and the full /actions page.
export function useFleetActions(limit = 50, pollMs = 15000) {
  const [actions, setActions] = useState<FleetAction[]>([]);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(() => {
    const gateways = getGateways();
    Promise.allSettled(gateways.map((gw) => new GatewayClient(gw).actions(limit))).then((results) => {
      const merged: FleetAction[] = [];
      results.forEach((r, i) => {
        if (r.status === "fulfilled") {
          merged.push(...r.value.map((a) => ({ ...a, site: gateways[i].name })));
        }
      });
      merged.sort((a, b) => (a.time < b.time ? 1 : -1));
      setActions(merged);
      setLoading(false);
    });
  }, [limit]);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, pollMs);
    return () => clearInterval(id);
  }, [refresh, pollMs]);

  return { actions, loading, refresh };
}
