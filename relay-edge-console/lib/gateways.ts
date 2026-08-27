import gatewaysConfig from "@/config/gateways.json";
import { Gateway } from "./types";

// The fleet roster: one entry per relay-edge gateway/site. Static, edited
// by hand and redeployed for Phase 1 (see plan: "no new Go service" — the
// frontend does fleet aggregation itself by fanning out to this list).
export function getGateways(): Gateway[] {
  return gatewaysConfig as Gateway[];
}

export function getGateway(name: string): Gateway | undefined {
  return getGateways().find((g) => g.name === name);
}
