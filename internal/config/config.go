package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Tenant       string        `json:"tenant"`
	Site         string        `json:"site"`
	Gateway      string        `json:"gateway"`
	Listen       string        `json:"listen"`
	SpoolDir     string        `json:"spool_dir"`
	ActionLogDir string        `json:"action_log_dir"`
	Relay        RelayConfig   `json:"relay"`
	Console      ConsoleConfig `json:"console"`
	TLS          TLSConfig     `json:"tls"`
	Devices      []Device      `json:"devices"`
}

// TLSConfig optionally terminates TLS directly on the API listener
// (relay-edge is deployed to arbitrary hosts, some without a reverse proxy
// already in place). Leave both empty for the default plain-HTTP listener.
type TLSConfig struct {
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`
}

// ConsoleConfig configures the operator-console API (/v1/console/*).
// A single shared bearer token is the Phase 1 auth model; see docs.
type ConsoleConfig struct {
	Token       string `json:"token,omitempty"`
	AllowOrigin string `json:"allow_origin,omitempty"`
}

type RelayConfig struct {
	Endpoint  string `json:"endpoint"`
	Token     string `json:"token,omitempty"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
}

type Device struct {
	ID         string   `json:"id"`
	Protocol   string   `json:"protocol"`
	Endpoint   string   `json:"endpoint"`
	IntervalMS int      `json:"interval_ms,omitempty"`
	Priority   int      `json:"priority,omitempty"`
	UnitID     byte     `json:"unit_id,omitempty"`
	Topic      string   `json:"topic,omitempty"`
	ClientID   string   `json:"client_id,omitempty"`
	Signals    []Signal `json:"signals,omitempty"`
}

type Signal struct {
	Name     string  `json:"name"`
	Kind     string  `json:"kind,omitempty"`
	Address  uint16  `json:"address,omitempty"`
	DataType string  `json:"data_type,omitempty"`
	Scale    float64 `json:"scale,omitempty"`
	Writable bool    `json:"writable,omitempty"`
	Topic    string  `json:"topic,omitempty"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if c.Listen == "" {
		c.Listen = ":8080"
	}
	if c.SpoolDir == "" {
		c.SpoolDir = "/var/lib/relay-edge/spool"
	}
	if c.ActionLogDir == "" {
		c.ActionLogDir = "/var/lib/relay-edge/actionlog"
	}
	if c.Relay.TimeoutMS == 0 {
		c.Relay.TimeoutMS = 3000
	}
	for i := range c.Devices {
		if c.Devices[i].IntervalMS == 0 {
			c.Devices[i].IntervalMS = 2000
		}
		if c.Devices[i].ClientID == "" {
			c.Devices[i].ClientID = "relay-edge-" + c.Devices[i].ID
		}
		if c.Devices[i].Priority == 0 {
			c.Devices[i].Priority = 2
		}
		for j := range c.Devices[i].Signals {
			if c.Devices[i].Signals[j].Scale == 0 {
				c.Devices[i].Signals[j].Scale = 1
			}
			if c.Devices[i].Signals[j].DataType == "" {
				c.Devices[i].Signals[j].DataType = "uint16"
			}
			if c.Devices[i].Signals[j].Kind == "" {
				c.Devices[i].Signals[j].Kind = "holding"
			}
		}
	}
	return c, nil
}
