FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/relay-edge ./cmd/relay-edge && \
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/relay-sim ./cmd/relay-sim && \
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/relay-mock ./cmd/relay-mock

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/relay-edge /relay-edge
ENTRYPOINT ["/relay-edge"]
