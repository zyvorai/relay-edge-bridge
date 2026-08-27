package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zyvorai/relay-edge/internal/actionlog"
	"github.com/zyvorai/relay-edge/internal/api"
	"github.com/zyvorai/relay-edge/internal/config"
	"github.com/zyvorai/relay-edge/internal/registry"
	"github.com/zyvorai/relay-edge/internal/spool"
	"github.com/zyvorai/relay-edge/pkg/model"
	"github.com/zyvorai/relay-edge/pkg/protocol"
	mb "github.com/zyvorai/relay-edge/pkg/protocol/modbus"
	mq "github.com/zyvorai/relay-edge/pkg/protocol/mqtt"
	relayclient "github.com/zyvorai/relay-edge/pkg/relay"
)

type Engine struct {
	cfg       config.Config
	queue     *spool.Queue
	relay     *relayclient.Client
	writers   map[string]protocol.Writer
	ctx       context.Context
	cancel    context.CancelFunc
	server    *http.Server
	wg        sync.WaitGroup
	accepted  atomic.Uint64
	forwarded atomic.Uint64
	failures  atomic.Uint64
	actions   atomic.Uint64

	registry  *registry.Registry
	actionLog *actionlog.Log
	bus       *api.Broadcaster
}

type Action struct {
	ID       string  `json:"id"`
	DeviceID string  `json:"device_id"`
	Target   string  `json:"target"`
	Value    float64 `json:"value"`
	Actor    string  `json:"actor,omitempty"`
}

func New(c config.Config) (*Engine, error) {
	q, err := spool.Open(c.SpoolDir)
	if err != nil {
		return nil, err
	}
	al, err := actionlog.Open(c.ActionLogDir)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		cfg: c, queue: q,
		relay:   relayclient.New(c.Relay.Endpoint, c.Relay.Token, time.Duration(c.Relay.TimeoutMS)*time.Millisecond),
		writers: map[string]protocol.Writer{},
		ctx:     ctx, cancel: cancel,
		registry:  registry.New(c.Devices),
		actionLog: al,
		bus:       api.NewBroadcaster(200),
	}
	return e, nil
}

func (e *Engine) Start() error {
	for _, d := range e.cfg.Devices {
		switch strings.ToLower(d.Protocol) {
		case "modbus", "modbus-tcp":
			a := mb.NewAdapter(d, e.cfg.Tenant, e.cfg.Site, e.cfg.Gateway)
			e.writers[d.ID] = a
			e.startPoller(d, a)
		case "mqtt":
			a := mq.NewAdapter(d, e.cfg.Tenant, e.cfg.Site, e.cfg.Gateway)
			e.writers[d.ID] = a
			e.startStreamer(d, a)
		default:
			log.Printf("device %s: unsupported protocol %s; HTTP ingest remains available", d.ID, d.Protocol)
		}
	}
	e.wg.Add(1)
	go e.forwardLoop()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", e.health)
	mux.HandleFunc("/readyz", e.health)
	mux.HandleFunc("/metrics", e.metrics)
	mux.HandleFunc("/v1/ingest/", e.ingest)
	mux.HandleFunc("/v1/actions", e.action)
	consoleAPI := &api.Server{
		Tenant: e.cfg.Tenant, Site: e.cfg.Site, Gateway: e.cfg.Gateway,
		Token:       e.cfg.Console.Token,
		AllowOrigin: e.cfg.Console.AllowOrigin,
		Registry:    e.registry,
		ActionLog:   e.actionLog,
		Bus:         e.bus,
		Counters: api.Counters{
			Accepted:  e.accepted.Load,
			Forwarded: e.forwarded.Load,
			Failures:  e.failures.Load,
			Actions:   e.actions.Load,
		},
		QueueStats:    e.queue.Stats,
		Devices:       e.cfg.Devices,
		ActionHandler: e.action,
	}
	consoleAPI.Routes(mux)
	e.server = &http.Server{Addr: e.cfg.Listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	useTLS := e.cfg.TLS.CertFile != "" && e.cfg.TLS.KeyFile != ""
	go func() {
		var err error
		if useTLS {
			log.Printf("relay-edge API listening on %s (TLS)", e.cfg.Listen)
			err = e.server.ListenAndServeTLS(e.cfg.TLS.CertFile, e.cfg.TLS.KeyFile)
		} else {
			log.Printf("relay-edge API listening on %s", e.cfg.Listen)
			err = e.server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("http server: %v", err)
			e.cancel()
		}
	}()
	return nil
}
func (e *Engine) Wait() { <-e.ctx.Done() }
func (e *Engine) Shutdown(ctx context.Context) error {
	e.cancel()
	if e.server != nil {
		_ = e.server.Shutdown(ctx)
	}
	done := make(chan struct{})
	go func() { e.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Engine) startPoller(d config.Device, p protocol.Poller) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		ticker := time.NewTicker(time.Duration(d.IntervalMS) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-e.ctx.Done():
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(e.ctx, time.Duration(d.IntervalMS)*time.Millisecond)
				ev, err := p.Poll(ctx)
				cancel()
				e.registry.MarkPoll(d.ID, ev, err)
				if err != nil {
					e.failures.Add(1)
					log.Printf("poll %s: %v", d.ID, err)
					continue
				}
				if err := e.Emit(ev); err != nil {
					log.Printf("emit %s: %v", d.ID, err)
				}
			}
		}
	}()
}
func (e *Engine) startStreamer(d config.Device, s protocol.Streamer) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		backoff := time.Second
		for {
			if e.ctx.Err() != nil {
				return
			}
			err := s.Run(e.ctx, func(ev model.Event) error {
				e.registry.MarkStream(d.ID, ev, nil)
				return e.Emit(ev)
			})
			if e.ctx.Err() != nil {
				return
			}
			e.registry.MarkStream(d.ID, model.Event{}, err)
			e.failures.Add(1)
			log.Printf("stream %s: %v; reconnecting in %s", d.ID, err, backoff)
			select {
			case <-e.ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
		}
	}()
}
func (e *Engine) Emit(ev model.Event) error {
	if err := ev.Validate(); err != nil {
		return err
	}
	if err := e.queue.Put(ev); err != nil {
		return err
	}
	e.accepted.Add(1)
	e.bus.Publish(ev)
	return nil
}
func (e *Engine) forwardLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < 100; i++ {
				name, ev, err := e.queue.Peek()
				if errors.Is(err, os.ErrNotExist) {
					break
				}
				if err != nil {
					e.failures.Add(1)
					break
				}
				ctx, cancel := context.WithTimeout(e.ctx, time.Duration(e.cfg.Relay.TimeoutMS)*time.Millisecond)
				err = e.relay.Send(ctx, ev)
				cancel()
				if err != nil {
					e.failures.Add(1)
					break
				}
				if err = e.queue.Ack(name); err != nil {
					e.failures.Add(1)
					break
				}
				e.forwarded.Add(1)
			}
		}
	}
}

func (e *Engine) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
func (e *Engine) metrics(w http.ResponseWriter, r *http.Request) {
	n, _ := e.queue.Len()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "relay_edge_events_accepted_total %d\nrelay_edge_events_forwarded_total %d\nrelay_edge_failures_total %d\nrelay_edge_actions_total %d\nrelay_edge_spool_depth %d\n", e.accepted.Load(), e.forwarded.Load(), e.failures.Load(), e.actions.Load(), n)
}
func (e *Engine) ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	device := strings.TrimPrefix(r.URL.Path, "/v1/ingest/")
	if device == "" || strings.Contains(device, "/") {
		http.Error(w, "invalid device", 400)
		return
	}
	var data any
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(&data); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	ev, err := model.New("relay-edge://"+e.cfg.Gateway+"/http/"+device, "io.relay.telemetry.http", "device/"+device, e.cfg.Tenant, e.cfg.Site, e.cfg.Gateway, 2, map[string]any{"protocol": "http", "payload": data})
	if err == nil {
		err = e.Emit(ev)
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(202)
	fmt.Fprintf(w, "{\"event_id\":%q}\n", ev.ID)
}
func (e *Engine) action(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var a Action
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&a); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if a.DeviceID == "" || a.Target == "" {
		http.Error(w, "device_id and target required", 400)
		return
	}
	wr, ok := e.writers[a.DeviceID]
	if !ok {
		http.Error(w, "device has no writable protocol adapter", 404)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	actionResult, err := wr.Write(ctx, a.Target, a.Value)
	cancel()
	status := actionResult.Status
	if status == "" {
		status = "executed"
	}
	if err != nil {
		status = "failed"
		e.failures.Add(1)
	} else {
		e.actions.Add(1)
	}
	payload := map[string]any{"action_id": a.ID, "device_id": a.DeviceID, "target": a.Target, "value": a.Value, "status": status, "verified": actionResult.Verified, "observed": actionResult.Observed, "detail": actionResult.Detail}
	result, _ := model.New("relay-edge://"+e.cfg.Gateway+"/actions", "io.relay.action."+status, "device/"+a.DeviceID, e.cfg.Tenant, e.cfg.Site, e.cfg.Gateway, 1, payload)
	_ = e.Emit(result)
	_ = e.actionLog.Append(actionlog.Record{
		ActionID: a.ID, DeviceID: a.DeviceID, Target: a.Target, Value: a.Value, Actor: a.Actor,
		Status: status, Verified: actionResult.Verified, Detail: actionResult.Detail, Time: result.Time,
	})
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "{\"status\":%q,\"verified\":%t,\"event_id\":%q}\n", status, actionResult.Verified, result.ID)
}
