package modbus

import (
	"context"
	sim "github.com/zyvorai/relay-edge/pkg/sim/modbus"
	"net"
	"testing"
	"time"
)

func TestClientServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := sim.New(addr)
	go func() { _ = s.Run(ctx) }()
	deadline := time.Now().Add(time.Second)
	for {
		c, err := net.DialTimeout("tcp", addr, 20*time.Millisecond)
		if err == nil {
			c.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cl := Client{Endpoint: addr, UnitID: 1, Timeout: time.Second}
	v, err := cl.ReadRegisters(context.Background(), false, 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 3 {
		t.Fatalf("len=%d", len(v))
	}
	if err := cl.WriteRegister(context.Background(), 10, 1500); err != nil {
		t.Fatal(err)
	}
	v, err = cl.ReadRegisters(context.Background(), false, 10, 1)
	if err != nil || v[0] != 1500 {
		t.Fatalf("v=%v err=%v", v, err)
	}
}
