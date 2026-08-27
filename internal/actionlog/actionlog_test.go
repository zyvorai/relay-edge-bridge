package actionlog

import "testing"

func TestLog(t *testing.T) {
	l, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{"a1", "a2", "a3"} {
		if err := l.Append(Record{ActionID: id, DeviceID: "pump-17", Target: "rpm_setpoint", Value: float64(i), Status: "verified", Verified: true}); err != nil {
			t.Fatal(err)
		}
	}
	recs, err := l.Recent(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
	if recs[0].ActionID != "a3" {
		t.Fatalf("expected most-recent-first, got %q", recs[0].ActionID)
	}

	if err := l.Prune(1); err != nil {
		t.Fatal(err)
	}
	recs, err = l.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].ActionID != "a3" {
		t.Fatalf("expected prune to keep only the newest record, got %+v", recs)
	}
}
