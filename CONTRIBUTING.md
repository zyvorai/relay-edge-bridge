# Contributing

Protocol adapters and simulators are welcome. New adapters should normalize to `pkg/model.Event`, avoid vendor lock-in, include tests or a simulator where feasible, and must not add protocol-specific assumptions to Relay Core.

Run `gofmt -w .` and `go test ./...` before submitting changes.
