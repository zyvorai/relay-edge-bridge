// Mirrors the JSON shapes served by relay-edge's /v1/console/* API
// (internal/api/handlers.go, internal/registry, internal/actionlog).

export type DeviceState = {
  id: string;
  protocol: string;
  connected: boolean;
  last_seen?: string;
  last_error?: string;
  last_values?: Record<string, unknown>;
  failures: number;
};

export type ActionRecord = {
  action_id: string;
  device_id: string;
  target: string;
  value: number;
  actor?: string;
  status: string;
  verified: boolean;
  detail?: string;
  time: string;
};

export type RelayEvent = {
  specversion: string;
  id: string;
  source: string;
  type: string;
  subject?: string;
  time: string;
  tenant?: string;
  site?: string;
  gateway?: string;
  priority?: number;
  data: unknown;
};

export type Summary = {
  tenant: string;
  site: string;
  gateway: string;
  device_count: number;
  healthy_count: number;
  spool_depth: number;
  accepted_total: number;
  forwarded_total: number;
  failures_total: number;
  actions_total: number;
};

export type SpoolStats = {
  depth: number;
  oldest_pending_age_seconds?: number;
  priorities?: Record<string, number>;
};

export type ConfigDevice = {
  id: string;
  protocol: string;
  endpoint: string;
  interval_ms?: number;
  priority?: number;
  topic?: string;
  signals?: { name: string; kind?: string; writable?: boolean }[];
};

export type Gateway = { name: string; url: string; token: string };
