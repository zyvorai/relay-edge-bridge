#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="${TMPDIR:-/tmp}/relay-edge-local-demo"
rm -rf "$TMP" "$ROOT/spool"
mkdir -p "$TMP"
cd "$ROOT"
go build -o "$TMP/relay-edge" ./cmd/relay-edge
go build -o "$TMP/relay-sim" ./cmd/relay-sim
go build -o "$TMP/relay-mock" ./cmd/relay-mock
"$TMP/relay-mock" -listen 127.0.0.1:18081 >"$TMP/mock.log" 2>&1 & P1=$!
"$TMP/relay-sim" -mode modbus -listen 127.0.0.1:11502 >"$TMP/modbus.log" 2>&1 & P2=$!
"$TMP/relay-sim" -mode mqtt-broker -listen 127.0.0.1:11883 >"$TMP/mqtt.log" 2>&1 & P3=$!
cleanup(){ kill "$P1" "$P2" "$P3" "${P4:-}" "${P5:-}" 2>/dev/null || true; }
trap cleanup EXIT INT TERM
sleep .4
"$TMP/relay-edge" -config examples/config.local.json >"$TMP/edge.log" 2>&1 & P4=$!
sleep .5
"$TMP/relay-sim" -mode mqtt-sensor -endpoint mqtt://127.0.0.1:11883 -topic factory/plant-01/pump-17/telemetry -device pump-17 -interval 300ms >"$TMP/sensor.log" 2>&1 & P5=$!
sleep 3
printf '%s\n' '--- metrics ---'
curl -fsS http://127.0.0.1:18080/metrics
printf '%s\n' '--- verified Modbus action ---'
curl -fsS -X POST http://127.0.0.1:18080/v1/actions -H 'content-type: application/json' -d '{"id":"demo-rpm-1","device_id":"pump-17","target":"rpm_setpoint","value":1450}'
printf '\n%s\n' '--- recent events ---'
sleep .8
curl -fsS http://127.0.0.1:18081/v1/events/recent
printf '\n'
