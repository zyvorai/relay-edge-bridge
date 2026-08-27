package modbussim

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"math"
	"net"
	"sync"
	"time"
)

type Server struct {
	Addr string
	mu   sync.RWMutex
	regs [1024]uint16
	ln   net.Listener
}

func New(addr string) *Server {
	s := &Server{Addr: addr}
	s.regs[0] = 720
	s.regs[1] = 135
	s.regs[2] = 1800
	s.regs[10] = 1800
	return s
}

func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.physics(ctx)
	go func() { <-ctx.Done(); _ = ln.Close() }()
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go s.handle(c)
	}
}

func (s *Server) physics(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	phase := 0.0
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.mu.Lock()
			phase += 0.2
			target := s.regs[10]
			rpm := s.regs[2]
			if rpm < target {
				rpm += min16(25, target-rpm)
			} else if rpm > target {
				rpm -= min16(25, rpm-target)
			}
			s.regs[2] = rpm
			temp := 72.0 + 4*math.Sin(phase) + float64(rpm-1500)/300.0
			pressure := 13.5 + 0.5*math.Sin(phase/2)
			s.regs[0] = uint16(math.Round(temp * 10))
			s.regs[1] = uint16(math.Round(pressure * 10))
			s.mu.Unlock()
		}
	}
}

func min16(a, b uint16) uint16 {
	if a < b {
		return a
	}
	return b
}
func (s *Server) handle(c net.Conn) {
	defer c.Close()
	for {
		hdr := make([]byte, 7)
		if _, err := io.ReadFull(c, hdr); err != nil {
			return
		}
		n := int(binary.BigEndian.Uint16(hdr[4:6])) - 1
		if n < 1 || n > 260 {
			return
		}
		pdu := make([]byte, n)
		if _, err := io.ReadFull(c, pdu); err != nil {
			return
		}
		resp := s.exec(pdu)
		out := make([]byte, 7+len(resp))
		copy(out[0:4], hdr[0:4])
		binary.BigEndian.PutUint16(out[4:6], uint16(len(resp)+1))
		out[6] = hdr[6]
		copy(out[7:], resp)
		if _, err := c.Write(out); err != nil {
			return
		}
	}
}
func (s *Server) exec(p []byte) []byte {
	if len(p) < 1 {
		return []byte{0x80, 3}
	}
	fc := p[0]
	switch fc {
	case 3, 4:
		if len(p) != 5 {
			return []byte{fc | 0x80, 3}
		}
		addr := binary.BigEndian.Uint16(p[1:3])
		qty := binary.BigEndian.Uint16(p[3:5])
		if qty == 0 || int(addr)+int(qty) > len(s.regs) {
			return []byte{fc | 0x80, 2}
		}
		out := make([]byte, 2+qty*2)
		out[0] = fc
		out[1] = byte(qty * 2)
		s.mu.RLock()
		for i := uint16(0); i < qty; i++ {
			binary.BigEndian.PutUint16(out[2+i*2:4+i*2], s.regs[addr+i])
		}
		s.mu.RUnlock()
		return out
	case 6:
		if len(p) != 5 {
			return []byte{fc | 0x80, 3}
		}
		addr := binary.BigEndian.Uint16(p[1:3])
		if int(addr) >= len(s.regs) {
			return []byte{fc | 0x80, 2}
		}
		v := binary.BigEndian.Uint16(p[3:5])
		s.mu.Lock()
		s.regs[addr] = v
		s.mu.Unlock()
		log.Printf("modbus-sim write register %d=%d", addr, v)
		return append([]byte(nil), p...)
	case 16:
		if len(p) < 6 {
			return []byte{fc | 0x80, 3}
		}
		addr := binary.BigEndian.Uint16(p[1:3])
		qty := binary.BigEndian.Uint16(p[3:5])
		bc := int(p[5])
		if qty == 0 || bc != int(qty)*2 || len(p) != 6+bc || int(addr)+int(qty) > len(s.regs) {
			return []byte{fc | 0x80, 3}
		}
		s.mu.Lock()
		for i := uint16(0); i < qty; i++ {
			s.regs[addr+i] = binary.BigEndian.Uint16(p[6+i*2 : 8+i*2])
		}
		s.mu.Unlock()
		out := make([]byte, 5)
		out[0] = 16
		copy(out[1:], p[1:5])
		return out
	default:
		return []byte{fc | 0x80, 1}
	}
}

var _ = errors.New
