package mqttsim

import "testing"

func TestMatch(t *testing.T) {
	if !match("factory/+/+/telemetry", "factory/a/b/telemetry") {
		t.Fatal("expected match")
	}
	if match("factory/+/+/telemetry", "factory/a/b/c/telemetry") {
		t.Fatal("unexpected match")
	}
}
