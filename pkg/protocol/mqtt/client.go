package mqtt

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"time"
)

type Message struct {
	Topic   string
	Payload []byte
}
type Client struct {
	Endpoint, ClientID string
	KeepAlive          time.Duration
	mu                 sync.Mutex
	conn               net.Conn
	r                  *bufio.Reader
	packetID           uint16
}

func (c *Client) dial(ctx context.Context) error {
	ep := c.Endpoint
	if c.ClientID == "" {
		c.ClientID = "relay-edge"
	}
	if c.KeepAlive == 0 {
		c.KeepAlive = 30 * time.Second
	}
	u, err := url.Parse(ep)
	if err != nil {
		return err
	}
	addr := u.Host
	if u.Scheme == "" {
		addr = ep
	}
	if addr == "" {
		return fmt.Errorf("bad mqtt endpoint %q", ep)
	}
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	c.conn = conn
	c.r = bufio.NewReader(conn)
	body := append(putString("MQTT"), 4, 2, 0, byte(int(c.KeepAlive.Seconds())))
	body = append(body, putString(c.ClientID)...)
	if err := writePacket(conn, 1, 0, body); err != nil {
		conn.Close()
		return err
	}
	p, err := readPacket(c.r)
	if err != nil {
		conn.Close()
		return err
	}
	if p.Type != 2 || len(p.Body) < 2 || p.Body[1] != 0 {
		conn.Close()
		return fmt.Errorf("mqtt connack rejected")
	}
	return nil
}
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
func (c *Client) nextID() uint16 {
	c.packetID++
	if c.packetID == 0 {
		c.packetID = 1
	}
	return c.packetID
}
func (c *Client) Subscribe(ctx context.Context, topic string, out chan<- Message) error {
	if err := c.dial(ctx); err != nil {
		return err
	}
	id := c.nextID()
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, id)
	b = append(b, putString(topic)...)
	b = append(b, 0)
	if err := writePacket(c.conn, 8, 2, b); err != nil {
		return err
	}
	p, err := readPacket(c.r)
	if err != nil {
		return err
	}
	if p.Type != 9 {
		return fmt.Errorf("expected suback")
	}
	go c.readLoop(ctx, out)
	return nil
}
func (c *Client) readLoop(ctx context.Context, out chan<- Message) {
	defer c.Close()
	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.KeepAlive))
		p, err := readPacket(c.r)
		if err != nil {
			return
		}
		switch p.Type {
		case 3:
			topic, rest, err := getString(p.Body)
			if err != nil {
				continue
			}
			select {
			case out <- Message{Topic: topic, Payload: append([]byte(nil), rest...)}:
			case <-ctx.Done():
				return
			}
		case 13:
		default:
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
func Publish(ctx context.Context, endpoint, clientID, topic string, payload []byte) error {
	c := &Client{Endpoint: endpoint, ClientID: clientID, KeepAlive: 30 * time.Second}
	if err := c.dial(ctx); err != nil {
		return err
	}
	defer c.Close()
	body := append(putString(topic), payload...)
	if dl, ok := ctx.Deadline(); ok {
		_ = c.conn.SetWriteDeadline(dl)
	}
	return writePacket(c.conn, 3, 0, body)
}

var _ = io.EOF
