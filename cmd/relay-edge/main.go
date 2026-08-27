package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zyvorai/relay-edge/internal/config"
	"github.com/zyvorai/relay-edge/internal/engine"
)

func main() {
	cfgPath := flag.String("config", "/etc/relay-edge/config.json", "config path")
	flag.Parse()
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	e, err := engine.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err = e.Start(); err != nil {
		log.Fatal(err)
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
	case <-time.After(365 * 24 * time.Hour):
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = e.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
