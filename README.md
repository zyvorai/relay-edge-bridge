# Relay Edge

**Protocol-neutral edge bridge for Relay.** Relay Edge converts physical and IoT protocols into a canonical CloudEvents-compatible envelope, durably spools events during WAN loss, forwards them to any Relay-compatible HTTP endpoint, and translates normalized actions back to devices.

The architectural boundary is deliberate:

- **Relay** stays generic: event durability, acknowledgement, routing, workflow, retry, replay and action lifecycle.
- **Relay Edge** owns edge/device protocol churn: Modbus, MQTT, HTTP, OPC-UA plugins, simulators and gateway deployment.
- Vendor products are integrations, not dependencies. No Armada, Siemens, Schneider, Rockwell, AWS, GCP or Zyvor proprietary API is required by the core.

## Why a separate project?

Yes. Putting Modbus/OPC-UA/MQTT directly into Relay would make Relay an IoT gateway and couple its release cycle to field protocols. `relay-edge` is the anti-corruption layer between the physical world and Relay.

```text
 PLC / sensor / camera / gateway
            |
   Modbus / MQTT / OPC-UA / HTTP
            |
       relay-edge
   normalize + spool + actions
            |
  CloudEvents-compatible HTTP
            |
          Relay
            |
 workflow / AI / humans / APIs
```

## Included in this repository

| Capability | Status | Notes |
|---|---|---|
| HTTP ingest | Working | Arbitrary JSON sensor/device payloads |
| Modbus TCP read | Working | FC3/FC4, typed/scaled registers |
| Modbus TCP write | Working | FC6 + read-after-write verification |
| MQTT 3.1.1 subscribe | Working | QoS 0, `+`/`#` broker filters |
| MQTT command publish | Working | QoS 0; correctly reported as executed, not physically verified |
| Durable store-and-forward | Working | Atomic filesystem spool, no database dependency |
| Modbus simulator | Working | Dynamic temperature/pressure/RPM + writable setpoint |
| MQTT broker simulator | Working | Small CI/dev broker supporting CONNECT/SUBSCRIBE/PUBLISH/PING |
| MQTT sensor simulator | Working | Dynamic industrial telemetry |
| HTTP sensor simulator | Working | Dynamic industrial telemetry |
| Kubernetes deployment | Included | PVC, probes, non-root, NetworkPolicy skeleton |
| systemd deployment | Included | Hardened unit for bare Linux gateways |
| OPC-UA | Optional plugin | Separate Go submodule using `gopcua` |

The base module is **stdlib-only** and builds without downloading dependencies.

## Quick start without Docker

Terminal 1:

```bash
go run ./cmd/relay-mock -listen :8081
```

Terminal 2:

```bash
go run ./cmd/relay-sim -mode modbus -listen :1502
```

Terminal 3:

```bash
go run ./cmd/relay-sim -mode mqtt-broker -listen :1883
```

For a host-only demo copy `examples/config.json` and change service names to `127.0.0.1`.

Then:

```bash
go run ./cmd/relay-edge -config ./config.local.json
```

Generate MQTT telemetry:

```bash
go run ./cmd/relay-sim \
  -mode mqtt-sensor \
  -endpoint mqtt://127.0.0.1:1883 \
  -topic factory/plant-01/pump-17/telemetry \
  -device pump-17
```

## Docker Compose demo

```bash
docker compose up --build
```

This starts:

- `relay-edge` on `:8080`
- a mock Relay sink on `:8081`
- Modbus simulator on `:1502`
- MQTT simulator broker on `:1883`
- an MQTT sensor publisher

Check metrics:

```bash
curl http://127.0.0.1:8080/metrics
```

Execute a Modbus setpoint action:

```bash
./scripts/action-demo.sh
```

Example response:

```json
{"status":"verified","verified":true,"event_id":"..."}
```

For Modbus, `verified=true` means the register was read back after the write and matched the requested value. It does **not** automatically mean the physical machine completed the real-world operation. Relay can layer a telemetry condition on top for physical verification.

## Canonical event

Example emitted event:

```json
{
  "specversion":"1.0",
  "id":"c9dd...",
  "source":"relay-edge://edge-gw-01/modbus/pump-17",
  "type":"io.relay.telemetry.modbus",
  "subject":"device/pump-17",
  "time":"2026-08-27T17:00:00Z",
  "datacontenttype":"application/json",
  "tenant":"acme",
  "site":"plant-01",
  "gateway":"edge-gw-01",
  "priority":1,
  "data":{
    "protocol":"modbus-tcp",
    "values":{"temperature_c":75.2,"pressure_bar":13.6,"rpm":1775}
  }
}
```

Relay should consume the envelope based on `type`, `subject`, extensions and `data`; it should never need protocol-specific code.

## Actions

Relay, an operator, or an agent can call:

```http
POST /v1/actions
Content-Type: application/json
```

```json
{
  "id":"action-123",
  "device_id":"pump-17",
  "target":"rpm_setpoint",
  "value":1450
}
```

The result is also emitted as an event:

- `io.relay.action.verified` — state could be read back and matched.
- `io.relay.action.executed` — command was handed to the protocol but physical state is not proven.
- `io.relay.action.failed` — adapter rejected or failed the command.

This distinction is important for autonomous/agentic operations.

## Configuration

The base project uses JSON intentionally so the executable remains dependency-free.

```json
{
  "tenant":"acme",
  "site":"plant-01",
  "gateway":"edge-gw-01",
  "listen":":8080",
  "spool_dir":"/var/lib/relay-edge/spool",
  "relay":{"endpoint":"https://relay.example.com/v1/events","token":"...","timeout_ms":3000},
  "devices":[]
}
```

See `examples/config.json` for Modbus and MQTT examples.

## Protocol design

New protocol packages implement one or more tiny interfaces in `pkg/protocol`:

```go
type Poller interface {
    Poll(context.Context) (model.Event, error)
}

type Streamer interface {
    Run(context.Context, func(model.Event) error) error
}

type Writer interface {
    Write(context.Context, string, float64) (ActionResult, error)
}
```

Recommended future open-source adapters:

- OPC-UA
- SNMP v2/v3
- CAN/CANopen
- BACnet/IP
- ONVIF camera events
- LoRaWAN Network Server webhooks
- NMEA / GPS
- serial/RS-485 adapters
- GPIO/IIO Linux sensors
- Kubernetes events and node telemetry
- Cilium/Hubble network events

Keep vendor-specific mappings in profiles/configuration wherever possible.

## Store and forward

Events are written atomically to `spool_dir` before forwarding. When Relay/WAN is unavailable the files remain on disk. The forward loop resumes in lexical insertion order when connectivity returns.

For very high-rate deployments, replace the spool package behind the same API with an embedded WAL such as Pebble/RocksDB or a site-local stream, but do not force those dependencies onto small gateways.

## Kubernetes

```bash
kubectl apply -k deploy/k8s
```

Before production, replace the sample Relay URL, device IPs and broad private-CIDR NetworkPolicy with site-specific values.

## Bare Linux / industrial PC

```bash
make linux-amd64
# or
make linux-arm64
```

Install the binary, create a `relay-edge` system user, put config at `/etc/relay-edge/config.json`, then install `deploy/systemd/relay-edge.service`.

## Security defaults

- Core container is non-root and supports read-only root filesystem.
- Device writes must be explicitly configured with `writable:true`.
- HTTP body sizes are bounded.
- Spool filenames are sanitized and atomically renamed.
- The sample NetworkPolicy is a starting point, not a finished plant policy.
- Production MQTT should add TLS/auth and QoS1 before critical control use.
- Production control actions should include authorization/policy, idempotency and safety interlocks in Relay.

## Build and test

```bash
make test
make build
```

The base repository currently passes:

```text
go test ./...
```

and has been exercised end-to-end with the included Modbus and MQTT simulators plus a Modbus write/readback action.

## Open-source model

Recommended license: **Apache-2.0** for the core, adapters and simulators. Keep enterprise fleet management, policy packs, signed device catalogs and long-term support in commercial products if desired, but make the wire formats and basic adapters open.

## Repository relationship

```text
relay
  generic durable event/action fabric

relay-edge
  generic edge/device protocol bridge (this repo)

relay-edge/plugins/opcua
  optional heavier protocol dependency
```

Do not rename `relay` into an IoT product. Its generality is the strategic advantage.

## Support

Free under Apache-2.0. If it helps you, you can donate any amount you like: <https://rzp.io/rzp/utwdYh3>. A donation is voluntary and supports development and maintenance. It is not a purchase: it buys no support, SLA, features, priority or license, and it is not tax-deductible (Zyvor AI Labs Private Limited is a company, not a charity). Support contracts are separate: [sales@zyvor.dev](mailto:sales@zyvor.dev).
