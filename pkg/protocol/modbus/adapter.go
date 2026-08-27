package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/zyvorai/relay-edge/internal/config"
	"github.com/zyvorai/relay-edge/pkg/model"
	"github.com/zyvorai/relay-edge/pkg/protocol"
)

type Adapter struct {
	Device                config.Device
	Tenant, Site, Gateway string
	client                Client
}

func NewAdapter(d config.Device, tenant, site, gateway string) *Adapter {
	return &Adapter{Device: d, Tenant: tenant, Site: site, Gateway: gateway, client: Client{Endpoint: d.Endpoint, UnitID: d.UnitID, Timeout: 2 * time.Second}}
}

func (a *Adapter) Poll(ctx context.Context) (model.Event, error) {
	values := map[string]any{}
	for _, s := range a.Device.Signals {
		words := uint16(1)
		switch strings.ToLower(s.DataType) {
		case "uint32", "int32", "float32":
			words = 2
		}
		regs, err := a.client.ReadRegisters(ctx, strings.EqualFold(s.Kind, "input"), s.Address, words)
		if err != nil {
			return model.Event{}, fmt.Errorf("%s: %w", s.Name, err)
		}
		v, err := decode(regs, s.DataType, s.Scale)
		if err != nil {
			return model.Event{}, err
		}
		values[s.Name] = v
	}
	data := map[string]any{"protocol": "modbus-tcp", "endpoint": a.Device.Endpoint, "unit_id": a.Device.UnitID, "values": values}
	return model.New("relay-edge://"+a.Gateway+"/modbus/"+a.Device.ID, "io.relay.telemetry.modbus", "device/"+a.Device.ID, a.Tenant, a.Site, a.Gateway, a.Device.Priority, data)
}

func (a *Adapter) Write(ctx context.Context, target string, value float64) (protocol.ActionResult, error) {
	for _, s := range a.Device.Signals {
		if s.Name != target || !s.Writable {
			continue
		}
		scale := s.Scale
		if scale == 0 {
			scale = 1
		}
		raw := value / scale
		var want uint16
		switch strings.ToLower(s.DataType) {
		case "uint16":
			want = uint16(math.Round(raw))
		case "int16":
			want = uint16(int16(math.Round(raw)))
		default:
			return protocol.ActionResult{}, fmt.Errorf("writable data type %s not implemented", s.DataType)
		}
		if err := a.client.WriteRegister(ctx, s.Address, want); err != nil {
			return protocol.ActionResult{}, err
		}
		// Read-after-write is a protocol-level verification. For a physical actuator,
		// applications may add a second telemetry condition in Relay before declaring
		// the real-world operation complete.
		regs, err := a.client.ReadRegisters(ctx, false, s.Address, 1)
		if err != nil {
			return protocol.ActionResult{Status: "executed", Detail: "write acknowledged; readback failed"}, nil
		}
		verified := len(regs) == 1 && regs[0] == want
		status := "executed"
		if verified {
			status = "verified"
		}
		return protocol.ActionResult{Status: status, Verified: verified, Observed: float64(regs[0]) * scale, Detail: "modbus register readback"}, nil
	}
	return protocol.ActionResult{}, fmt.Errorf("writable target %q not found", target)
}

func decode(regs []uint16, typ string, scale float64) (any, error) {
	if scale == 0 {
		scale = 1
	}
	switch strings.ToLower(typ) {
	case "uint16":
		return float64(regs[0]) * scale, nil
	case "int16":
		return float64(int16(regs[0])) * scale, nil
	case "uint32":
		if len(regs) < 2 {
			return nil, fmt.Errorf("need 2 registers")
		}
		return float64(uint32(regs[0])<<16|uint32(regs[1])) * scale, nil
	case "int32":
		if len(regs) < 2 {
			return nil, fmt.Errorf("need 2 registers")
		}
		u := uint32(regs[0])<<16 | uint32(regs[1])
		return float64(int32(u)) * scale, nil
	case "float32":
		if len(regs) < 2 {
			return nil, fmt.Errorf("need 2 registers")
		}
		b := make([]byte, 4)
		binary.BigEndian.PutUint16(b[0:2], regs[0])
		binary.BigEndian.PutUint16(b[2:4], regs[1])
		return float64(math.Float32frombits(binary.BigEndian.Uint32(b))) * scale, nil
	default:
		return nil, fmt.Errorf("unsupported data type %q", typ)
	}
}
