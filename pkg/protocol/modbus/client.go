package modbus

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"
)

type Client struct {
	Endpoint string
	UnitID   byte
	Timeout  time.Duration
	tx       uint32
}

func (c *Client) transact(ctx context.Context, pdu []byte) ([]byte, error) {
	d := net.Dialer{Timeout: c.Timeout}
	conn, err := d.DialContext(ctx, "tcp", c.Endpoint)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else if c.Timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(c.Timeout))
	}
	tx := uint16(atomic.AddUint32(&c.tx, 1))
	hdr := make([]byte, 7)
	binary.BigEndian.PutUint16(hdr[0:2], tx)
	binary.BigEndian.PutUint16(hdr[2:4], 0)
	binary.BigEndian.PutUint16(hdr[4:6], uint16(len(pdu)+1))
	hdr[6] = c.UnitID
	if _, err = conn.Write(append(hdr, pdu...)); err != nil {
		return nil, err
	}
	rh := make([]byte, 7)
	if _, err = io.ReadFull(conn, rh); err != nil {
		return nil, err
	}
	if binary.BigEndian.Uint16(rh[0:2]) != tx {
		return nil, fmt.Errorf("transaction id mismatch")
	}
	n := int(binary.BigEndian.Uint16(rh[4:6])) - 1
	if n <= 0 || n > 260 {
		return nil, fmt.Errorf("invalid response length %d", n)
	}
	out := make([]byte, n)
	if _, err = io.ReadFull(conn, out); err != nil {
		return nil, err
	}
	if out[0]&0x80 != 0 {
		if len(out) < 2 {
			return nil, fmt.Errorf("modbus exception")
		}
		return nil, fmt.Errorf("modbus exception function=%d code=%d", out[0]&0x7f, out[1])
	}
	return out, nil
}

func (c *Client) ReadRegisters(ctx context.Context, input bool, address, quantity uint16) ([]uint16, error) {
	if quantity == 0 || quantity > 125 {
		return nil, fmt.Errorf("invalid quantity %d", quantity)
	}
	fc := byte(3)
	if input {
		fc = 4
	}
	p := []byte{fc, 0, 0, 0, 0}
	binary.BigEndian.PutUint16(p[1:3], address)
	binary.BigEndian.PutUint16(p[3:5], quantity)
	r, err := c.transact(ctx, p)
	if err != nil {
		return nil, err
	}
	if len(r) < 2 || r[0] != fc {
		return nil, fmt.Errorf("invalid modbus response")
	}
	count := int(r[1])
	if count != int(quantity)*2 || len(r) != count+2 {
		return nil, fmt.Errorf("unexpected byte count")
	}
	vals := make([]uint16, quantity)
	for i := range vals {
		vals[i] = binary.BigEndian.Uint16(r[2+i*2 : 4+i*2])
	}
	return vals, nil
}

func (c *Client) WriteRegister(ctx context.Context, address, value uint16) error {
	p := []byte{6, 0, 0, 0, 0}
	binary.BigEndian.PutUint16(p[1:3], address)
	binary.BigEndian.PutUint16(p[3:5], value)
	r, err := c.transact(ctx, p)
	if err != nil {
		return err
	}
	if len(r) != 5 || r[0] != 6 {
		return fmt.Errorf("invalid write response")
	}
	return nil
}

func (c *Client) WriteRegisters(ctx context.Context, address uint16, values []uint16) error {
	if len(values) == 0 || len(values) > 123 {
		return fmt.Errorf("invalid values count")
	}
	p := make([]byte, 6+len(values)*2)
	p[0] = 16
	binary.BigEndian.PutUint16(p[1:3], address)
	binary.BigEndian.PutUint16(p[3:5], uint16(len(values)))
	p[5] = byte(len(values) * 2)
	for i, v := range values {
		binary.BigEndian.PutUint16(p[6+i*2:8+i*2], v)
	}
	r, err := c.transact(ctx, p)
	if err != nil {
		return err
	}
	if len(r) != 5 || r[0] != 16 {
		return fmt.Errorf("invalid multi-write response")
	}
	return nil
}
