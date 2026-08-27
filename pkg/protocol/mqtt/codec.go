package mqtt

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

type Packet struct {
	Type  byte
	Flags byte
	Body  []byte
}

func readPacket(r *bufio.Reader) (Packet, error) {
	h, err := r.ReadByte()
	if err != nil {
		return Packet{}, err
	}
	n, err := readRemaining(r)
	if err != nil {
		return Packet{}, err
	}
	if n > 4*1024*1024 {
		return Packet{}, fmt.Errorf("mqtt packet too large")
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return Packet{}, err
	}
	return Packet{Type: h >> 4, Flags: h & 0x0f, Body: b}, nil
}
func writePacket(w io.Writer, typ, flags byte, body []byte) error {
	h := []byte{typ<<4 | flags}
	h = append(h, encodeRemaining(len(body))...)
	_, err := w.Write(append(h, body...))
	return err
}
func encodeRemaining(n int) []byte {
	var out []byte
	for {
		d := byte(n % 128)
		n /= 128
		if n > 0 {
			d |= 0x80
		}
		out = append(out, d)
		if n == 0 {
			return out
		}
	}
}
func readRemaining(r *bufio.Reader) (int, error) {
	m := 1
	v := 0
	for i := 0; i < 4; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		v += int(b&127) * m
		if b&128 == 0 {
			return v, nil
		}
		m *= 128
	}
	return 0, fmt.Errorf("malformed remaining length")
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
	if len(b) < 2+n {
		return "", nil, io.ErrUnexpectedEOF
	}
	return string(b[2 : 2+n]), b[2+n:], nil
}
func topicMatch(filter, topic string) bool {
	fi, ti := splitTopic(filter), splitTopic(topic)
	for i := 0; i < len(fi); i++ {
		if fi[i] == "#" {
			return i == len(fi)-1
		}
		if i >= len(ti) {
			return false
		}
		if fi[i] != "+" && fi[i] != ti[i] {
			return false
		}
	}
	return len(fi) == len(ti)
}
func splitTopic(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '/' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}
