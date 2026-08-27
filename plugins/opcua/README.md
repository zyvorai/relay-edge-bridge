# relay-edge OPC-UA plugin

Optional Go submodule for environments that need OPC-UA. It intentionally lives outside the dependency-free core because OPC-UA brings a much larger protocol/security implementation and a faster-changing API surface.

Build with network access:

```bash
cd plugins/opcua
go mod tidy
go build ./cmd/relay-opcua
go build ./cmd/opcua-sim
```

Run simulator:

```bash
./opcua-sim -host 0.0.0.0 -port 4840
```

Bridge selected nodes into relay-edge:

```bash
./relay-opcua \
  -endpoint opc.tcp://127.0.0.1:4840 \
  -edge http://127.0.0.1:8080 \
  -device compressor-17 \
  -node temperature='ns=2;s=Temperature' \
  -node pressure='ns=2;s=Pressure'
```

This plugin is Apache-2.0; its upstream `gopcua` dependency is MIT licensed.
