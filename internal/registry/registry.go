// Package registry tracks per-device connection health and last-known
// values (a "digital shadow") for the operator console. It is in-memory
// only and intentionally not persisted: this state is inherently ephemeral
// ("what is the last known value right now") and repopulates within one
// poll/stream cycle after a restart, so durability would add disk I/O to
// the hottest-churn state for no real benefit.
package registry

import (
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/zyvorai/relay-edge/internal/config"
	"github.com/zyvorai/relay-edge/pkg/model"
)

type DeviceState struct {
	ID         string         `json:"id"`
	Protocol   string         `json:"protocol"`
	Connected  bool           `json:"connected"`
	LastSeen   time.Time      `json:"last_seen,omitempty"`
	LastError  string         `json:"last_error,omitempty"`
	LastValues map[string]any `json:"last_values,omitempty"`
	Failures   uint64         `json:"failures"`
}

type Registry struct {
	mu     sync.RWMutex
	states map[string]*DeviceState
}

// New seeds the registry with every configured device so they appear
// (as disconnected, with zero values) even before their first poll/stream
// tick completes.
func New(devices []config.Device) *Registry {
	r := &Registry{states: make(map[string]*DeviceState, len(devices))}
	for _, d := range devices {
		r.states[d.ID] = &DeviceState{ID: d.ID, Protocol: d.Protocol}
	}
	return r
}

// MarkPoll records the outcome of one poll-style adapter tick (Modbus, HTTP).
func (r *Registry) MarkPoll(id string, ev model.Event, err error) { r.mark(id, ev, err) }

// MarkStream records the outcome of one streaming-adapter delivery (MQTT).
func (r *Registry) MarkStream(id string, ev model.Event, err error) { r.mark(id, ev, err) }

func (r *Registry) mark(id string, ev model.Event, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.states[id]
	if !ok {
		s = &DeviceState{ID: id}
		r.states[id] = s
	}
	if err != nil {
		s.Connected = false
		s.LastError = err.Error()
		s.Failures++
		return
	}
	s.Connected = true
	s.LastError = ""
	s.LastSeen = time.Now().UTC()
	if s.Protocol == "" {
		s.Protocol = protocolFromData(ev.Data)
	}
	if vals := extractValues(ev.Data); vals != nil {
		s.LastValues = vals
	}
}

// Snapshot returns a stable, independently-mutable copy of every known
// device's current state, sorted by ID for deterministic API responses.
func (r *Registry) Snapshot() []DeviceState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]DeviceState, 0, len(r.states))
	for _, s := range r.states {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns one device's current state.
func (r *Registry) Get(id string) (DeviceState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.states[id]
	if !ok {
		return DeviceState{}, false
	}
	return *s, true
}

// extractValues pulls the digital-shadow payload out of an event's Data.
// Protocol adapters nest sampled points under a "values" object
// (pkg/protocol/modbus/adapter.go, pkg/protocol/mqtt/adapter.go); fall back
// to the whole decoded object for producers that don't follow that shape
// (e.g. generic HTTP ingest).
func extractValues(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	if v, ok := m["values"].(map[string]any); ok {
		return v
	}
	return m
}

func protocolFromData(raw json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	p, _ := m["protocol"].(string)
	return p
}
