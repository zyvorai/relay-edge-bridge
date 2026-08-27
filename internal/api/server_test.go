package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zyvorai/relay-edge/internal/actionlog"
	"github.com/zyvorai/relay-edge/internal/config"
	"github.com/zyvorai/relay-edge/internal/registry"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	al, err := actionlog.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &Server{
		Tenant: "acme", Site: "plant-01", Gateway: "gw-01",
		Token:       "secret",
		AllowOrigin: "http://localhost:3000",
		Registry:    registry.New([]config.Device{{ID: "pump-17", Protocol: "modbus-tcp"}}),
		ActionLog:   al,
		Bus:         NewBroadcaster(10),
		Counters: Counters{
			Accepted: func() uint64 { return 1 }, Forwarded: func() uint64 { return 1 },
			Failures: func() uint64 { return 0 }, Actions: func() uint64 { return 0 },
		},
	}
}

func TestAuthRequired(t *testing.T) {
	mux := http.NewServeMux()
	testServer(t).Routes(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/console/devices", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a token, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/console/devices", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with a valid token, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCORSPreflight(t *testing.T) {
	mux := http.NewServeMux()
	testServer(t).Routes(mux)

	req := httptest.NewRequest(http.MethodOptions, "/v1/console/summary", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
}

func TestSummary(t *testing.T) {
	mux := http.NewServeMux()
	testServer(t).Routes(mux)
	req := httptest.NewRequest(http.MethodGet, "/v1/console/summary", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
