package spool

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zyvorai/relay-edge/pkg/model"
)

type Queue struct {
	dir string
	mu  sync.Mutex
}

func Open(dir string) (*Queue, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return &Queue{dir: dir}, nil
}

func (q *Queue) Put(e model.Event) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	priority := e.Priority
	if priority < 0 {
		priority = 0
	}
	if priority > 99 {
		priority = 99
	}
	name := fmt.Sprintf("%02d-%020d-%s.event", priority, time.Now().UnixNano(), sanitize(e.ID))
	tmp := filepath.Join(q.dir, "."+name+".tmp")
	dst := filepath.Join(q.dir, name)
	if err := os.WriteFile(tmp, append(b, '\n'), 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func (q *Queue) Peek() (string, model.Event, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	ents, err := os.ReadDir(q.dir)
	if err != nil {
		return "", model.Event{}, err
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".event") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", model.Event{}, os.ErrNotExist
	}
	sort.Strings(names)
	path := filepath.Join(q.dir, names[0])
	b, err := os.ReadFile(path)
	if err != nil {
		return "", model.Event{}, err
	}
	var ev model.Event
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&ev); err != nil {
		return "", model.Event{}, err
	}
	return names[0], ev, nil
}

func (q *Queue) Ack(name string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return errors.New("invalid spool name")
	}
	return os.Remove(filepath.Join(q.dir, name))
}

func (q *Queue) Len() (int, error) {
	ents, err := os.ReadDir(q.dir)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".event") {
			n++
		}
	}
	return n, nil
}

// Stats reports queue depth, the age of the oldest pending event, and a
// per-priority breakdown, for offline/store-and-forward visibility in the
// operator console. It parses the existing "%02d-%020d-<id>.event" filename
// encoding (see Put) rather than opening every file.
type Stats struct {
	Depth             int            `json:"depth"`
	OldestPendingAgeS float64        `json:"oldest_pending_age_seconds,omitempty"`
	Priorities        map[string]int `json:"priorities,omitempty"`
}

func (q *Queue) Stats() (Stats, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	ents, err := os.ReadDir(q.dir)
	if err != nil {
		return Stats{}, err
	}
	st := Stats{Priorities: map[string]int{}}
	oldest := int64(math.MaxInt64)
	for _, e := range ents {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".event") {
			continue
		}
		st.Depth++
		parts := strings.SplitN(name, "-", 3)
		if len(parts) < 2 {
			continue
		}
		st.Priorities[parts[0]]++
		if ts, err := strconv.ParseInt(parts[1], 10, 64); err == nil && ts < oldest {
			oldest = ts
		}
	}
	if st.Depth > 0 && oldest != int64(math.MaxInt64) {
		st.OldestPendingAgeS = time.Since(time.Unix(0, oldest)).Seconds()
	}
	return st, nil
}

func sanitize(s string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", "..", "_", ":", "_")
	return r.Replace(s)
}
