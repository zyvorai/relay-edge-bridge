package registry

import (
	"errors"
	"testing"

	"github.com/zyvorai/relay-edge/internal/config"
	"github.com/zyvorai/relay-edge/pkg/model"
)

func TestRegistry(t *testing.T) {
	r := New([]config.Device{{ID: "pump-17", Protocol: "modbus-tcp"}})

	snap := r.Snapshot()
	if len(snap) != 1 || snap[0].Connected {
		t.Fatalf("expected one seeded, disconnected device, got %+v", snap)
	}

	ev, err := model.New("test://x", "test.event", "device/pump-17", "t", "s", "g", 1,
		map[string]any{"protocol": "modbus-tcp", "values": map[string]any{"temperature_c": 75.2}})
	if err != nil {
		t.Fatal(err)
	}
	r.MarkPoll("pump-17", ev, nil)

	got, ok := r.Get("pump-17")
	if !ok {
		t.Fatal("expected pump-17 to exist")
	}
	if !got.Connected {
		t.Fatal("expected connected=true after successful poll")
	}
	if got.LastValues["temperature_c"] != 75.2 {
		t.Fatalf("expected digital shadow to carry temperature_c, got %+v", got.LastValues)
	}

	r.MarkPoll("pump-17", model.Event{}, errors.New("dial tcp: timeout"))
	got, _ = r.Get("pump-17")
	if got.Connected {
		t.Fatal("expected connected=false after failed poll")
	}
	if got.Failures != 1 {
		t.Fatalf("expected failures=1, got %d", got.Failures)
	}
	if got.LastError == "" {
		t.Fatal("expected last_error to be set")
	}
}
