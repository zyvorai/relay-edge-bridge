// Package actionlog persists a durable local history of device actions.
// Today, an action's outcome is only ever spooled as an event and the
// spool file is deleted once forwarded (internal/spool.Queue.Ack) — nothing
// local survives after that. actionlog closes that gap using the exact
// atomic write-then-rename idiom internal/spool already uses, so the
// operator console can show "what did we command and was it verified"
// even when the upstream Relay backend is unreachable.
package actionlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Record struct {
	ActionID string    `json:"action_id"`
	DeviceID string    `json:"device_id"`
	Target   string    `json:"target"`
	Value    float64   `json:"value"`
	Actor    string    `json:"actor,omitempty"`
	Status   string    `json:"status"`
	Verified bool      `json:"verified"`
	Detail   string    `json:"detail,omitempty"`
	Time     time.Time `json:"time"`
}

type Log struct {
	dir string
	mu  sync.Mutex
}

func Open(dir string) (*Log, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return &Log{dir: dir}, nil
}

// Append durably records one action outcome. The filename is zero-padded
// by nanosecond timestamp so a directory listing sorts newest-last, the
// same trick internal/spool uses for priority ordering.
func (l *Log) Append(r Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if r.Time.IsZero() {
		r.Time = time.Now().UTC()
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%020d-%s.json", r.Time.UnixNano(), sanitize(r.ActionID))
	tmp := filepath.Join(l.dir, "."+name+".tmp")
	dst := filepath.Join(l.dir, name)
	if err := os.WriteFile(tmp, append(b, '\n'), 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// Recent returns up to limit records, most recent first.
func (l *Log) Recent(limit int) ([]Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ents, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if limit > 0 && len(names) > limit {
		names = names[:limit]
	}
	out := make([]Record, 0, len(names))
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(l.dir, n))
		if err != nil {
			continue
		}
		var r Record
		if err := json.NewDecoder(bytes.NewReader(b)).Decode(&r); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// Prune deletes the oldest records beyond keep, bounding disk usage for a
// gateway that runs indefinitely without an operator ever rotating logs.
func (l *Log) Prune(keep int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	ents, err := os.ReadDir(l.dir)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	if len(names) <= keep {
		return nil
	}
	sort.Strings(names)
	for _, n := range names[:len(names)-keep] {
		_ = os.Remove(filepath.Join(l.dir, n))
	}
	return nil
}

func sanitize(s string) string {
	if s == "" {
		s = "unknown"
	}
	r := strings.NewReplacer("/", "_", "\\", "_", "..", "_", ":", "_")
	return r.Replace(s)
}
