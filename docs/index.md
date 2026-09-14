# relay-edge-bridge

**Protocol-neutral edge bridge for [Zyvor Relay](https://github.com/zyvorai/relay).** It converts physical and IoT protocols into a canonical CloudEvents-compatible envelope, durably spools events during WAN loss, forwards them to any Relay-compatible HTTP endpoint, and translates normalized actions back to devices.

Apache-2.0 · [source on GitHub](https://github.com/zyvorai/relay-edge-bridge)

## Boundary

- **Relay** stays generic: event durability, acknowledgement, routing, workflow, retry, replay, and action lifecycle.
- **relay-edge-bridge** owns edge/device protocol churn: Modbus, MQTT, HTTP, optional OPC-UA plugins, simulators, and gateway deployment.
- Vendor products are integrations, not dependencies.

```text
 PLC / sensor / camera / gateway
            |
   Modbus / MQTT / OPC-UA / HTTP
            |
   relay-edge-bridge
   normalize + spool + actions
            |
  CloudEvents-compatible HTTP
            |
          Relay
            |
 workflow / AI / humans / APIs
```

## Capabilities

| Capability | Status | Notes |
|---|---|---|
| HTTP ingest | Working | Arbitrary JSON sensor/device payloads |
| Modbus TCP read | Working | FC3/FC4, typed/scaled registers |
| Modbus TCP write | Working | FC6 + read-after-write verification |
| MQTT 3.1.1 subscribe | Working | QoS 0, `+`/`#` broker filters |
| MQTT command publish | Working | QoS 0; reported as executed, not physically verified |
| Durable store-and-forward | Working | Atomic filesystem spool, no database |
| Simulators | Working | Modbus, MQTT broker/sensor, HTTP sensor |
| Kubernetes / systemd | Included | PVC, probes, non-root, hardened unit |

The base module is **stdlib-only**.

## Quick start

```bash
go run ./cmd/relay-mock -listen :8081
go run ./cmd/relay-sim -mode modbus -listen :1502
go run ./cmd/relay-edge -config ./config.local.json
```

See the [README](https://github.com/zyvorai/relay-edge-bridge#readme) for host-only config, MQTT, Docker Compose, and Kubernetes.

## Next

- [Architecture](architecture.md) — invariants, data path, action path, verification levels
- [relay-edge](https://github.com/zyvorai/relay-edge) — site topology and simulators
- [relay-pubsub](https://github.com/zyvorai/relay-pubsub) — Pub/Sub gateway for Relay
