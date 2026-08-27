package api

import (
	"sync"

	"github.com/zyvorai/relay-edge/pkg/model"
)

// Broadcaster fans out accepted events (telemetry + action outcomes) to any
// number of live SSE subscribers, and retains a bounded in-memory backlog
// so a freshly-opened console page can render recent history before its
// stream connects. It is deliberately not persisted: relay-edge already
// durably queues every event in the spool for delivery to Relay: this is
// only a short live-view window for the console UI.
type Broadcaster struct {
	mu      sync.Mutex
	subs    map[chan model.Event]struct{}
	history []model.Event
	cap     int
}

// NewBroadcaster creates a broadcaster retaining up to historyCap recent
// events for GET /v1/console/events backlog requests.
func NewBroadcaster(historyCap int) *Broadcaster {
	if historyCap <= 0 {
		historyCap = 200
	}
	return &Broadcaster{subs: make(map[chan model.Event]struct{}), cap: historyCap}
}

// Publish records ev in the backlog and delivers it to every live
// subscriber. Slow subscribers never block ingestion: a subscriber whose
// buffered channel is full simply misses the event rather than stalling
// Publish (and, transitively, the engine's Emit path).
func (b *Broadcaster) Publish(ev model.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.history = append(b.history, ev)
	if len(b.history) > b.cap {
		b.history = b.history[len(b.history)-b.cap:]
	}
	for ch := range b.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Subscribe registers a new live listener. Call cancel when done to release
// the channel; the returned channel is buffered so a momentary consumer
// stall doesn't drop the very next event.
func (b *Broadcaster) Subscribe() (<-chan model.Event, func()) {
	ch := make(chan model.Event, 32)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	cancel := func() {
		b.mu.Lock()
		delete(b.subs, ch)
		b.mu.Unlock()
	}
	return ch, cancel
}

// Recent returns up to limit backlog events newer than since (zero value
// for "no lower bound"), oldest first.
func (b *Broadcaster) Recent(since func(model.Event) bool, limit int) []model.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]model.Event, 0, len(b.history))
	for _, ev := range b.history {
		if since == nil || since(ev) {
			out = append(out, ev)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}
