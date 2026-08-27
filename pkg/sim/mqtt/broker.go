package mqttsim

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type peer struct {
	conn    net.Conn
	r       *bufio.Reader
	mu      sync.Mutex
	filters []string
}
type Broker struct {
	Addr  string
	mu    sync.RWMutex
	peers map[*peer]struct{}
	ln    net.Listener
}

func New(addr string) *Broker { return &Broker{Addr: addr, peers: map[*peer]struct{}{}} }
func (b *Broker) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", b.Addr)
	if err != nil {
		return err
	}
	b.ln = ln
	go func() { <-ctx.Done(); _ = ln.Close() }()
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		p := &peer{conn: c, r: bufio.NewReader(c)}
		b.mu.Lock()
		b.peers[p] = struct{}{}
		b.mu.Unlock()
		go b.handle(p)
	}
}
func (b *Broker) handle(p *peer) {
	defer func() { p.conn.Close(); b.mu.Lock(); delete(b.peers, p); b.mu.Unlock() }()
	for {
		typ, flags, body, err := readPacket(p.r)
		if err != nil {
			return
		}
		switch typ {
		case 1:
			p.write(2, 0, []byte{0, 0})
		case 8:
			if len(body) < 2 {
				return
			}
			id := binary.BigEndian.Uint16(body[:2])
			rest := body[2:]
			var codes []byte
			for len(rest) > 0 {
				f, r, err := getString(rest)
				if err != nil {
					return
				}
				if len(r) < 1 {
					return
				}
				p.filters = append(p.filters, f)
				codes = append(codes, 0)
				rest = r[1:]
			}
			ack := []byte{byte(id >> 8), byte(id)}
			ack = append(ack, codes...)
			p.write(9, 0, ack)
		case 3:
			topic, payload, err := getString(body)
			if err != nil {
				return
			}
			log.Printf("mqtt-sim publish %s %s", topic, string(payload))
			b.broadcast(topic, payload)
		case 12:
			p.write(13, 0, nil)
		case 14:
			return
		default:
			_ = flags
		}
	}
}
func (b *Broker) broadcast(topic string, payload []byte) {
	b.mu.RLock()
	ps := make([]*peer, 0, len(b.peers))
	for p := range b.peers {
		ps = append(ps, p)
	}
	b.mu.RUnlock()
	body := append(putString(topic), payload...)
	for _, p := range ps {
		for _, f := range p.filters {
			if match(f, topic) {
				_ = p.write(3, 0, body)
				break
			}
		}
	}
}
func (p *peer) write(t, f byte, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return writePacket(p.conn, t, f, body)
}
func readPacket(r *bufio.Reader) (byte, byte, []byte, error) {
	h, e := r.ReadByte()
	if e != nil {
		return 0, 0, nil, e
	}
	n, e := readRemaining(r)
	if e != nil {
		return 0, 0, nil, e
	}
	body := make([]byte, n)
	_, e = io.ReadFull(r, body)
	return h >> 4, h & 15, body, e
}
func writePacket(w io.Writer, t, f byte, body []byte) error {
	buf := []byte{t<<4 | f}
	buf = append(buf, enc(len(body))...)
	buf = append(buf, body...)
	_, e := w.Write(buf)
	return e
}
func enc(n int) []byte {
	var o []byte
	for {
		d := byte(n % 128)
		n /= 128
		if n > 0 {
			d |= 128
		}
		o = append(o, d)
		if n == 0 {
			return o
		}
	}
}
func readRemaining(r *bufio.Reader) (int, error) {
	v, m := 0, 1
	for i := 0; i < 4; i++ {
		x, e := r.ReadByte()
		if e != nil {
			return 0, e
		}
		v += int(x&127) * m
		if x&128 == 0 {
			return v, nil
		}
		m *= 128
	}
	return 0, io.ErrUnexpectedEOF
}
func putString(s string) []byte {
	b := make([]byte, 2+len(s))
	binary.BigEndian.PutUint16(b[:2], uint16(len(s)))
	copy(b[2:], s)
	return b
}
func getString(b []byte) (string, []byte, error) {
	if len(b) < 2 {
		return "", nil, io.ErrUnexpectedEOF
	}
	n := int(binary.BigEndian.Uint16(b[:2]))
	if len(b) < n+2 {
		return "", nil, io.ErrUnexpectedEOF
	}
	return string(b[2 : n+2]), b[n+2:], nil
}
func match(filter, topic string) bool {
	f := strings.Split(filter, "/")
	t := strings.Split(topic, "/")
	for i := range f {
		if f[i] == "#" {
			return i == len(f)-1
		}
		if i >= len(t) {
			return false
		}
		if f[i] != "+" && f[i] != t[i] {
			return false
		}
	}
	return len(f) == len(t)
}
