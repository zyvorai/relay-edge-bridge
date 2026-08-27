"use client";

import { useCallback, useEffect, useState } from "react";
import { GatewayClient } from "@/lib/api-client";
import { getGateways } from "@/lib/gateways";
import { Summary } from "@/lib/types";

export type SiteSummary = {
  name: string;
  url: string;
  status: "online" | "unreachable";
  summary?: Summary;
  error?: string;
};

export type FleetSummary = {
  sites: SiteSummary[];
  totalDevices: number;
  healthyCount: number;
  healthyPct: number;
  degradedSites: number;
  loading: boolean;
  refresh: () => void;
};

// Fleet-wide aggregation lives entirely here, client-side: one settled
// fetch per configured gateway, merged. One unreachable gateway degrades
// gracefully (its tile reads "unreachable") instead of breaking the page.
export function useFleetSummary(pollMs = 15000): FleetSummary {
  const [sites, setSites] = useState<SiteSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(() => {
    const gateways = getGateways();
    Promise.allSettled(gateways.map((gw) => new GatewayClient(gw).summary())).then((results) => {
      setSites(
        results.map((r, i) => {
          const gw = gateways[i];
          if (r.status === "fulfilled") {
            return { name: gw.name, url: gw.url, status: "online" as const, summary: r.value };
          }
          return { name: gw.name, url: gw.url, status: "unreachable" as const, error: String(r.reason) };
        }),
      );
      setLoading(false);
    });
  }, []);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, pollMs);
    return () => clearInterval(id);
  }, [refresh, pollMs]);

  const totalDevices = sites.reduce((n, s) => n + (s.summary?.device_count ?? 0), 0);
  const healthyCount = sites.reduce((n, s) => n + (s.summary?.healthy_count ?? 0), 0);
  const degradedSites = sites.filter((s) => s.status === "unreachable").length;
  const healthyPct = totalDevices > 0 ? (healthyCount / totalDevices) * 100 : 100;

  return { sites, totalDevices, healthyCount, healthyPct, degradedSites, loading, refresh };
}
