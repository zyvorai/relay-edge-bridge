package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zyvorai/relay-edge/pkg/model"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	devices := s.Registry.Snapshot()
	healthy := 0
	for _, d := range devices {
		if d.Connected {
			healthy++
		}
	}
	depth := 0
	if s.QueueStats != nil {
		if st, err := s.QueueStats(); err == nil {
			depth = st.Depth
		}
	}
	writeJSON(w, map[string]any{
		"tenant":          s.Tenant,
		"site":            s.Site,
		"gateway":         s.Gateway,
		"device_count":    len(devices),
		"healthy_count":   healthy,
		"spool_depth":     depth,
		"accepted_total":  s.Counters.Accepted(),
		"forwarded_total": s.Counters.Forwarded(),
		"failures_total":  s.Counters.Failures(),
		"actions_total":   s.Counters.Actions(),
	})
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{"devices": s.Registry.Snapshot()})
}

func (s *Server) handleDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/console/devices/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "invalid device id", http.StatusBadRequest)
		return
	}
	dev, ok := s.Registry.Get(id)
	if !ok {
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}
	var recent []any
	if s.ActionLog != nil {
		if recs, err := s.ActionLog.Recent(200); err == nil {
			for _, rec := range recs {
				if rec.DeviceID == id {
					recent = append(recent, rec)
					if len(recent) >= 20 {
						break
					}
				}
			}
		}
	}
	writeJSON(w, map[string]any{"device": dev, "recent_actions": recent})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	var sinceFilter func(model.Event) bool
	if v := r.URL.Query().Get("since"); v != "" {
		if since, err := time.Parse(time.RFC3339, v); err == nil {
			sinceFilter = func(ev model.Event) bool { return ev.Time.After(since) }
		}
	}
	writeJSON(w, map[string]any{"events": s.Bus.Recent(sinceFilter, limit)})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "event: ready\ndata: {}\n\n")
	flusher.Flush()

	ch, cancel := s.Bus.Subscribe()
	defer cancel()

	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			b, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			evtType := "telemetry"
			if strings.HasPrefix(ev.Type, "io.relay.action.") {
				evtType = "action"
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evtType, b)
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleActions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := 100
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		recs, err := s.ActionLog.Recent(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"actions": recs})
	case http.MethodPost:
		if s.ActionHandler == nil {
			http.Error(w, "actions not configured", http.StatusNotImplemented)
			return
		}
		s.ActionHandler(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSpool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.QueueStats == nil {
		writeJSON(w, map[string]any{"depth": 0})
		return
	}
	st, err := s.QueueStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, st)
}

func (s *Server) handleConfigDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{"devices": s.Devices})
}
