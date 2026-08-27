# Architecture

## Invariants

1. Relay Core has no imports from device protocol packages.
2. Every inbound protocol is normalized before it crosses the edge boundary.
3. Every event is durably staged before WAN forwarding.
4. A command is not called `verified` merely because it was sent.
5. Protocol plugins are replaceable and vendor-neutral.
6. Local operation must continue when the upstream Relay endpoint is unavailable.

## Data path

```text
source -> adapter -> Event -> durable spool -> HTTP -> Relay
```

## Action path

```text
Relay/operator -> relay-edge /v1/actions -> Writer -> device
                                         -> action result Event -> spool -> Relay
```

## Verification levels

- `failed`: protocol operation failed.
- `executed`: command was accepted/sent, but target state was not observed.
- `verified`: the desired protocol/device state was observed after execution.
- physical/process verification should generally be a Relay workflow condition built from independent telemetry.

## Scaling

One relay-edge process per site/gateway is the default. For large plants, shard by protocol/network/security zone. Do not use active-active replicas against the same Modbus devices unless ownership/leader election is added, because duplicate polling/writes can be unsafe.
