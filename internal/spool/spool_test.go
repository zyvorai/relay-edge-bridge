package spool

import (
	"github.com/zyvorai/relay-edge/pkg/model"
	"testing"
)

func TestQueue(t *testing.T) {
	q, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e, err := model.New("test://x", "test.event", "device/x", "t", "s", "g", 1, map[string]any{"x": 1})
	if err != nil {
		t.Fatal(err)
	}
	if err = q.Put(e); err != nil {
		t.Fatal(err)
	}
	if n, _ := q.Len(); n != 1 {
		t.Fatalf("len=%d", n)
	}
	name, got, err := q.Peek()
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != e.ID {
		t.Fatalf("id mismatch")
	}
	if err = q.Ack(name); err != nil {
		t.Fatal(err)
	}
	if n, _ := q.Len(); n != 0 {
		t.Fatalf("len=%d", n)
	}
}
