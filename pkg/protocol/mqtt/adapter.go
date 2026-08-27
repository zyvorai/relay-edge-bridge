package mqtt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zyvorai/relay-edge/internal/config"
	"github.com/zyvorai/relay-edge/pkg/model"
	"github.com/zyvorai/relay-edge/pkg/protocol"
)

type Adapter struct {
	Device                config.Device
	Tenant, Site, Gateway string
}

func NewAdapter(d config.Device, tenant, site, gateway string) *Adapter {
	return &Adapter{Device: d, Tenant: tenant, Site: site, Gateway: gateway}
}
func (a *Adapter) Run(ctx context.Context, emit func(model.Event) error) error {
	c := &Client{Endpoint: a.Device.Endpoint, ClientID: a.Device.ClientID}
	ch := make(chan Message, 64)
	if err := c.Subscribe(ctx, a.Device.Topic, ch); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case m := <-ch:
			var payload any
			if json.Valid(m.Payload) {
				_ = json.Unmarshal(m.Payload, &payload)
			} else {
				payload = string(m.Payload)
			}
			e, err := model.New("relay-edge://"+a.Gateway+"/mqtt/"+a.Device.ID, "io.relay.telemetry.mqtt", "device/"+a.Device.ID, a.Tenant, a.Site, a.Gateway, a.Device.Priority, map[string]any{"protocol": "mqtt", "topic": m.Topic, "payload": payload})
			if err == nil {
				if err = emit(e); err != nil {
					return err
				}
			}
		}
	}
}
func (a *Adapter) Write(ctx context.Context, target string, value float64) (protocol.ActionResult, error) {
	topic := ""
	for _, s := range a.Device.Signals {
		if s.Name == target && s.Writable {
			topic = s.Topic
			break
		}
	}
	if topic == "" {
		return protocol.ActionResult{}, fmt.Errorf("writable mqtt target %q not found", target)
	}
	b, _ := json.Marshal(map[string]any{"target": target, "value": value})
	if err := Publish(ctx, a.Device.Endpoint, a.Device.ClientID+"-action", topic, b); err != nil {
		return protocol.ActionResult{}, err
	}
	// QoS0 publish means sent to the broker, not physical state verified.
	return protocol.ActionResult{Status: "executed", Verified: false, Detail: "mqtt command published; configure an acknowledgement telemetry rule in Relay for physical verification"}, nil
}
