// Package api implements the operator-console HTTP surface (/v1/console/*)
// for a relay-edge gateway: device health/digital-shadow, a live SSE event
// feed, action history, and offline/store-and-forward visibility. It is
// intentionally separate from internal/engine so that package doesn't grow
// any denser, and it never imports engine (engine imports this package).
package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/zyvorai/relay-edge/internal/actionlog"
	"github.com/zyvorai/relay-edge/internal/config"
	"github.com/zyvorai/relay-edge/internal/registry"
	"github.com/zyvorai/relay-edge/internal/spool"
)

// Counters exposes the engine's process-wide atomic counters without this
// package importing engine.
type Counters struct {
	Accepted  func() uint64
	Forwarded func() uint64
	Failures  func() uint64
	Actions   func() uint64
}

type Server struct {
	Tenant, Site, Gateway string
	Token                 string // required Phase 1 bearer token; empty means "deny everything"
	AllowOrigin           string // CORS origin allowed to call this API; empty disables CORS headers

	Registry   *registry.Registry
	ActionLog  *actionlog.Log
	Bus        *Broadcaster
	Counters   Counters
	QueueStats func() (spool.Stats, error)
	Devices    []config.Device

	// ActionHandler is the engine's existing POST /v1/actions handler,
	// reused as-is so action execution logic lives in exactly one place.
	ActionHandler http.HandlerFunc
}

// Routes registers every /v1/console/* endpoint on mux, each wrapped with
// CORS handling and bearer-token auth.
func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/console/summary", s.wrap(s.handleSummary))
	mux.HandleFunc("/v1/console/devices", s.wrap(s.handleDevices))
	mux.HandleFunc("/v1/console/devices/", s.wrap(s.handleDevice))
	mux.HandleFunc("/v1/console/events", s.wrap(s.handleEvents))
	mux.HandleFunc("/v1/console/stream", s.wrap(s.handleStream))
	mux.HandleFunc("/v1/console/actions", s.wrap(s.handleActions))
	mux.HandleFunc("/v1/console/spool", s.wrap(s.handleSpool))
	mux.HandleFunc("/v1/console/config/devices", s.wrap(s.handleConfigDevices))
}

// wrap applies CORS (including OPTIONS preflight short-circuit) then auth,
// in that order: a browser's preflight OPTIONS request carries no
// Authorization header, so it must be answered before authn runs.
func (s *Server) wrap(h http.HandlerFunc) http.HandlerFunc {
	return s.withCORS(s.authn(h))
}

func (s *Server) withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.AllowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.AllowOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *Server) authn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if s.Token == "" || token == "" ||
			subtle.ConstantTimeCompare([]byte(token), []byte(s.Token)) != 1 {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
