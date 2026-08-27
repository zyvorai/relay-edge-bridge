import {
  ActionRecord,
  ConfigDevice,
  DeviceState,
  Gateway,
  RelayEvent,
  SpoolStats,
  Summary,
} from "./types";

export class ApiError extends Error {
  constructor(
    public gateway: string,
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

// Thin per-gateway fetch wrapper for relay-edge's /v1/console/* API
// (internal/api/handlers.go). One instance per configured gateway — the
// frontend never talks to a shared backend, only directly to each edge.
export class GatewayClient {
  constructor(public gw: Gateway) {}

  private async request<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(`${this.gw.url}${path}`, {
      ...init,
      headers: { Authorization: `Bearer ${this.gw.token}`, ...init?.headers },
      cache: "no-store",
    });
    const body = await res.json().catch(() => ({}));
    if (!res.ok) {
      throw new ApiError(this.gw.name, res.status, body?.error ?? `${path}: HTTP ${res.status}`);
    }
    return body as T;
  }

  summary() {
    return this.request<Summary>("/v1/console/summary");
  }

  devices() {
    return this.request<{ devices: DeviceState[] }>("/v1/console/devices").then((r) => r.devices);
  }

  device(id: string) {
    return this.request<{ device: DeviceState; recent_actions: ActionRecord[] }>(
      `/v1/console/devices/${encodeURIComponent(id)}`,
    );
  }

  events(limit = 200) {
    return this.request<{ events: RelayEvent[] }>(`/v1/console/events?limit=${limit}`).then(
      (r) => r.events,
    );
  }

  actions(limit = 100) {
    return this.request<{ actions: ActionRecord[] }>(`/v1/console/actions?limit=${limit}`).then(
      (r) => r.actions,
    );
  }

  spool() {
    return this.request<SpoolStats>("/v1/console/spool");
  }

  configDevices() {
    return this.request<{ devices: ConfigDevice[] }>("/v1/console/config/devices").then(
      (r) => r.devices,
    );
  }

  postAction(body: { id: string; device_id: string; target: string; value: number; actor?: string }) {
    return this.request<{ status: string; verified: boolean; event_id: string }>(
      "/v1/console/actions",
      { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body) },
    );
  }

  streamUrl() {
    return `${this.gw.url}/v1/console/stream`;
  }
}
