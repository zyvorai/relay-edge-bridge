BINDIR ?= bin
GO ?= go

.PHONY: all build test fmt demo clean linux-amd64 linux-arm64
all: test build

build:
	mkdir -p $(BINDIR)
	CGO_ENABLED=0 $(GO) build -trimpath -o $(BINDIR)/relay-edge ./cmd/relay-edge
	CGO_ENABLED=0 $(GO) build -trimpath -o $(BINDIR)/relay-sim ./cmd/relay-sim
	CGO_ENABLED=0 $(GO) build -trimpath -o $(BINDIR)/relay-mock ./cmd/relay-mock

test:
	$(GO) test ./...

fmt:
	gofmt -w cmd internal pkg

linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -trimpath -o $(BINDIR)/relay-edge-linux-amd64 ./cmd/relay-edge

linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GO) build -trimpath -o $(BINDIR)/relay-edge-linux-arm64 ./cmd/relay-edge

demo:
	docker compose up --build

clean:
	rm -rf $(BINDIR)
