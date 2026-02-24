// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can

import (
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
)

// Message Flags.
type Flags int

const (
	// message type
	ExtFrame Flags = 1 << iota
	RTRMsg
	StatusMsg

	// FD specific flags
	FDSwitchBitrate
	ForceFD

	// if StatusMsg is set:
	MissingAck
	ErrorActive
	ErrorWarning
	ErrorPassive
	BusOff
	DataOverrun
	ReceiveBufferOverflow
)

// Reports wether the message is a status message, not a data message.
// In the first case, Msg fields Id, Len and Data should not be interpreted.
func (f Flags) IsStatus() bool {
	return f&StatusMsg != 0
}

// Reports whether the message contains an 29 bit wide, extended indentifier,
// or a standard 11 bit wide identifier.
func (f Flags) ExtFrame() bool {
	return f&ExtFrame != 0
}

func (f Flags) Test(t Flags) bool {
	return (f & t) == t
}

// Definition of a CAN Message.
type Msg struct {
	Id uint32 // The CAN message identifier
	Flags

	// data contains the payload of the CAN message.
	// It can be accessed through Data() and SetData()
	buf        DataBuffer
	stdPayload stdPayload

	Rx struct {
		Time Time // Timestamp
	}
}

func (m *Msg) Release() {
	if m.buf == nil {
		return
	}
	m.buf.Put()
	m.buf = nil
}

type stdPayload struct {
	n    uint8
	data [8]byte
}

func (p *stdPayload) Data() []byte {
	return p.data[:p.n]
}
func (p *stdPayload) Set(b []byte) {
	n := len(b)
	if n == 0 {
		p.Reset()
		return
	}
	if &b[0] != &p.data[0] {
		copy(p.data[:], b)
		if n > cap(p.data) {
			n = cap(p.data)
		}
	}
	p.n = uint8(n)
}

func (p *stdPayload) Put()   {}
func (p *stdPayload) Reset() { p.n = 0 }

type DataBuffer interface {
	// Data returns a byte slice with len set to the current data
	// portion, and cap set to the size of the underlying buffer.
	Data() []byte

	// Set updates the length of the current data if the underlying
	// buffer is the same. Otherwise it will use copy() to import
	// the specified bytes.
	Set([]byte)

	// Put returns the buffer back to its internally referenced pool.
	// Depending on whether a pool is associated, this may be a no-op.
	Put()

	// Reset sets the current slice to an empty slice.
	Reset()
}

// Data returns the current Payload of the message.
// If no payload buffer has been set by calling SetData before,
// Data returns a byte slice with the standard payload length 8.
func (m *Msg) Data() []byte {
	if m.buf == nil {
		return m.stdPayload.data[:m.stdPayload.n]
	}
	return m.buf.Data()
}

// SetData updates the payload of the message.
func (m *Msg) SetData(b []byte) {
	n := len(b)
	if n == 0 {
		m.stdPayload.n = 0
		if m.buf == nil {
			return
		}
		m.buf.Reset()
		return
	}
	p0 := &b[0]
	if p0 == &m.stdPayload.data[0] {
		m.stdPayload.Set(b)
		return
	}
	if m.buf != nil {
		m.buf.Put()
	}
	pd := PlainData(b)
	m.buf = &pd
}

// Attach is similar to SetData, but instead of a byte slice,
// a [DataBuffer] must be provided. This helps to avoid an internal
// allocation in case the data contains more than eight bytes.
func (m *Msg) Attach(b DataBuffer) {
	if m.buf != nil {
		m.buf.Put()
	}
	m.buf = b
}

// Import copies b into the message's backing store,
// either the standard payload array (if ≤ 8 bytes),
// or a user provided buffer previously set using SetData,
// if available. Else it will try to get a sufficient buffer
// from the pool, link it to the message, and copy the
// contents of b there. If the pool argument is nil,
// ErrMsgCapExceeded will be replied.
func (m *Msg) Import(b []byte, pool DataBufPool) error {
	n := len(b)
	if m.buf != nil {
		if n <= cap(m.buf.Data()) {
			m.buf.Set(b)
			return nil
		}
		m.buf.Put()
		m.buf = nil
	}
	if n <= cap(m.stdPayload.data) {
		m.stdPayload.Set(b)
		m.buf = &m.stdPayload
		return nil
	}
	if pool == nil {
		return ErrMsgCapExceeded
	}
	buf := pool.Get(n)
	buf.Set(b)
	m.buf = buf
	return nil
}

type PlainData []byte

func (pd PlainData) Data() []byte {
	return pd
}

func (pd *PlainData) Set(b []byte) {
	n := len(b)
	if n == 0 {
		pd.Reset()
		return
	}
	buf := *pd
	if n > cap(buf) {
		n = cap(buf)
	}
	buf = buf[:n]

	if &b[0] != &buf[0] {
		copy(buf, b)
	}
	*pd = buf
}

func (PlainData) Put() {}
func (pd *PlainData) Reset() {
	*pd = (*pd)[:0]
}

// Reset sets the message back to the initial state.
func (m *Msg) Reset() {
	m.Id = 0
	m.Flags = 0
	if m.buf == nil {
		return
	}
	m.buf.Reset()
}

var ValidFDSizes = []int{12, 16, 20, 24, 36, 48, 64}

func VerifyDataLenFD(n int) (next int, needsFD bool, err error) {
	if n <= 8 {
		return n, false, nil
	}

	needsFD = true
	for _, nFD := range ValidFDSizes {
		if n == nFD {
			return n, needsFD, nil
		}
		if n < nFD {
			return nFD, needsFD, ErrInvalidMsgLen
		}
	}
	return 0, needsFD, ErrInvalidMsgLen
}

var ErrInvalidMsgLen = errors.New("invalid message length")

var ErrMsgCapExceeded = errors.New("message capacity too small")

// FromExpr parses a CAN message expression string and stores the
// result into m, which may be a pre-initialized value.
// The format is similar to the format used by cansend from can-utils.
//
// CAN ID and data, separated by '#' or ':', must be specified in
// hexadecimal format. An FD frame can be forced using a double separator,
// followed by a CAN flags hex nibble; supported FD flags: BRS = 0b0001.
//
// The string may not contain white-space, but '.' can be used to
// separate data bytes.
func (m *Msg) FromExpr(expr string) error {
	if i := strings.IndexAny(expr, "#:"); i != -1 {
		sep := expr[i]
		sID := expr[:i]
		if sID != "" {
			if len(sID) > 3 {
				m.Flags |= ExtFrame
			}
			id, err := strconv.ParseUint(sID, 16, 32)
			if err != nil {
				return err
			}
			m.Id = uint32(id)
		}

		expr = expr[i+1:]
		if len(expr) == 0 {
			return nil
		}
		if expr[0] == sep {
			m.Flags |= ForceFD
			expr = expr[1:]
			if expr == "" {
				return errors.New("CAN FD flags value missing")
			}
			u, err := strconv.ParseInt(expr[:1], 16, 8)
			if err != nil {
				return err
			}
			if u&1 != 0 {
				m.Flags |= FDSwitchBitrate
			}
			expr = expr[1:]
		}
		if expr == "R" {
			m.Flags |= RTRMsg
			return nil
		}
	}
	expr = dots.Replace(expr)
	data := m.Data()
	n := hex.DecodedLen(len(expr))
	if n > cap(data) {
		data = make([]byte, n)
	} else {
		data = data[:n]
	}
	_, err := hex.Decode(data, []byte(expr))
	if err != nil {
		return err
	}
	m.SetData(data)
	return nil
}

var dots = strings.NewReplacer(".", "")

type DataBufPool interface {
	Get(minSize int) DataBuffer
	Put(DataBuffer)
}

type simpleBufPool struct {
	c chan *simpleDataBuf
}

func newSimpleBufPool(size int) *simpleBufPool {
	p := &simpleBufPool{c: make(chan *simpleDataBuf, size)}
	for range size / 2 {
		p.c <- p.allocBuf()
	}
	return p
}

func (p *simpleBufPool) Get(n int) DataBuffer {
	select {
	case buf, ok := <-p.c:
		if !ok {
			panic("pool channel closed unexpectedly")
		}

		if cap(buf.PlainData) < n {
			buf.PlainData = make([]byte, 0, n*3/2)
		}
		if buf.pool == nil {
			buf.pool = p
		}
		return buf
	default:
	}
	return p.allocBuf()
}

func (p *simpleBufPool) allocBuf() *simpleDataBuf {
	buf := new(simpleDataBuf)
	buf.PlainData = make([]byte, 0, 64)
	buf.pool = p
	return buf
}

func (p *simpleBufPool) Put(buf DataBuffer) {
	sb, ok := buf.(*simpleDataBuf)
	if !ok {
		return
	}
	select {
	case p.c <- sb:
	default:
	}
}

type simpleDataBuf struct {
	PlainData
	pool DataBufPool
}

func (sd *simpleDataBuf) Put() {
	sd.pool.Put(sd)
}
