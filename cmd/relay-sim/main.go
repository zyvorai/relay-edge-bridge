package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mq "github.com/zyvorai/relay-edge/pkg/protocol/mqtt"
	modbussim "github.com/zyvorai/relay-edge/pkg/sim/modbus"
	mqttsim "github.com/zyvorai/relay-edge/pkg/sim/mqtt"
)

func main() {
	mode := flag.String("mode", "modbus", "modbus|mqtt-broker|mqtt-sensor|http-sensor")
	listen := flag.String("listen", ":1502", "listen address")
	endpoint := flag.String("endpoint", "mqtt://127.0.0.1:1883", "target endpoint")
	topic := flag.String("topic", "factory/plant-01/pump-17/telemetry", "MQTT topic")
	device := flag.String("device", "pump-17", "device id")
	interval := flag.Duration("interval", time.Second, "publish interval")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	switch *mode {
	case "modbus":
		log.Printf("modbus simulator listening %s", *listen)
		if err := modbussim.New(*listen).Run(ctx); err != nil {
			log.Fatal(err)
		}
	case "mqtt-broker":
		log.Printf("mqtt simulator broker listening %s", *listen)
		if err := mqttsim.New(*listen).Run(ctx); err != nil {
			log.Fatal(err)
		}
	case "mqtt-sensor":
		runMQTTSensor(ctx, *endpoint, *topic, *device, *interval)
	case "http-sensor":
		runHTTPSensor(ctx, *endpoint, *device, *interval)
	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}
func sample(phase float64, device string) map[string]any {
	return map[string]any{"device_id": device, "temperature_c": 72 + 5*math.Sin(phase), "pressure_bar": 13.5 + 0.5*math.Sin(phase/2), "rpm": 1800 + int(120*math.Sin(phase/3)), "vibration_mm_s": 2.1 + 0.4*math.Sin(phase/4), "ts": time.Now().UTC()}
}
func runMQTTSensor(ctx context.Context, endpoint, topic, device string, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	phase := 0.0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			phase += 0.2
			b, _ := json.Marshal(sample(phase, device))
			cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := mq.Publish(cctx, endpoint, "relay-sim-"+device, topic, b)
			cancel()
			if err != nil {
				log.Printf("publish: %v", err)
			} else {
				log.Printf("mqtt -> %s %s", topic, b)
			}
		}
	}
}
func runHTTPSensor(ctx context.Context, endpoint, device string, interval time.Duration) {
	url := endpoint
	if len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	url += "/v1/ingest/" + device
	t := time.NewTicker(interval)
	defer t.Stop()
	phase := 0.0
	client := &http.Client{Timeout: 3 * time.Second}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			phase += 0.2
			b, _ := json.Marshal(sample(phase, device))
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("post: %v", err)
				continue
			}
			resp.Body.Close()
			log.Printf("http -> %s status=%s", url, resp.Status)
		}
	}
}

var _ = fmt.Sprintf
