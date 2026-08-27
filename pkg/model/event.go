package model

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Event is the canonical, protocol-neutral envelope emitted by relay-edge.
// It intentionally mirrors the useful CloudEvents 1.0 fields without requiring
// an SDK dependency, keeping the edge binary small and portable.
type Event struct {
	SpecVersion string          `json:"specversion"`
	ID          string          `json:"id"`
	Source      string          `json:"source"`
	Type        string          `json:"type"`
	Subject     string          `json:"subject,omitempty"`
	Time        time.Time       `json:"time"`
	DataSchema  string          `json:"dataschema,omitempty"`
	DataType    string          `json:"datacontenttype,omitempty"`
	Tenant      string          `json:"tenant,omitempty"`
	Site        string          `json:"site,omitempty"`
	Gateway     string          `json:"gateway,omitempty"`
	Priority    int             `json:"priority,omitempty"`
	TraceID     string          `json:"trace_id,omitempty"`
	Data        json.RawMessage `json:"data"`
}

func New(source, typ, subject, tenant, site, gateway string, priority int, data any) (Event, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Event{}, err
	}
	return Event{
		SpecVersion: "1.0", ID: newID(), Source: source, Type: typ, Subject: subject,
		Time: time.Now().UTC(), DataType: "application/json", Tenant: tenant,
		Site: site, Gateway: gateway, Priority: priority, Data: raw,
	}, nil
}

func (e Event) Validate() error {
	if e.SpecVersion != "1.0" {
		return fmt.Errorf("unsupported specversion %q", e.SpecVersion)
	}
	if e.ID == "" || e.Source == "" || e.Type == "" {
		return fmt.Errorf("id, source and type are required")
	}
	if e.Time.IsZero() {
		return fmt.Errorf("time is required")
	}
	if len(e.Data) == 0 {
		return fmt.Errorf("data is required")
	}
	return nil
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	// UUIDv4 shape without pulling a UUID dependency.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	x := hex.EncodeToString(b)
	return x[0:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:32]
}
