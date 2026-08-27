package protocol

import (
	"context"
	"github.com/zyvorai/relay-edge/pkg/model"
)

// Poller is implemented by request/response industrial protocols such as Modbus.
type Poller interface {
	Poll(context.Context) (model.Event, error)
}

// Streamer is implemented by subscription protocols such as MQTT.
type Streamer interface {
	Run(context.Context, func(model.Event) error) error
}

type ActionResult struct {
	Status   string `json:"status"` // executed or verified
	Verified bool   `json:"verified"`
	Observed any    `json:"observed,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// Writer executes a normalized Relay action against a device protocol.
// A protocol must not claim Verified unless it can read/observe the desired state.
type Writer interface {
	Write(context.Context, string, float64) (ActionResult, error)
}
