package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/zyvorai/relay-edge/pkg/model"
	"log"
	"net/http"
	"sync"
)

func main() {
	listen := flag.String("listen", ":8081", "listen address")
	flag.Parse()
	var mu sync.Mutex
	events := make([]model.Event, 0, 1024)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", 405)
			return
		}
		var e model.Event
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		mu.Lock()
		events = append(events, e)
		if len(events) > 1000 {
			events = events[len(events)-1000:]
		}
		mu.Unlock()
		log.Printf("event %-32s %-24s %s", e.Type, e.Subject, string(e.Data))
		w.WriteHeader(202)
	})
	mux.HandleFunc("/v1/events/recent", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(events)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok\n") })
	log.Printf("relay-mock listening %s", *listen)
	log.Fatal(http.ListenAndServe(*listen, mux))
}
