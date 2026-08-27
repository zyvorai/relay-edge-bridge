package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zyvorai/relay-edge/pkg/model"
)

type Client struct {
	endpoint, token string
	http            *http.Client
}

func New(endpoint, token string, timeout time.Duration) *Client {
	return &Client{endpoint: endpoint, token: token, http: &http.Client{Timeout: timeout}}
}

func (c *Client) Send(ctx context.Context, e model.Event) error {
	if c.endpoint == "" {
		return fmt.Errorf("relay endpoint is empty")
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/cloudevents+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("relay returned %s: %s", resp.Status, string(body))
	}
	return nil
}
