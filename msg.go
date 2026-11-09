// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Flags describe the format of a frame.
//
// The values are part of canrpc's wire format: never renumber them.
// Bits 2 and 5 to 11 carried status flags before v0.6.
type Flags int

const (
	ExtFrame Flags = 1 << 0 // 29-bit identifier
	RTRMsg   Flags = 1 << 1 // remote frame (classic CAN only)

	// CAN FD: bits of an FD frame's control field. On transmit,
	// FDF selects the FD format, which a payload longer than eight
	// bytes implies. On receive, drivers set FDF for every FD frame.
	BRS Flags = 1 << 3 // bit rate switch
	FDF Flags = 1 << 4 // FD format

	// ESI is the error state indicator of an FD frame: the sender was
	// error passive. Drivers report it on receive; it is not sent.
	ESI Flags = 1 << 12
)

// Earlier names of the FD flags.
const (
	// Deprecated: Use BRS.
	FDSwitchBitrate = BRS

	// Deprecated: Use FDF. Note that drivers set FDF on every
	// received FD frame, not only on frames of up to eight bytes.
	ForceFD = FDF
)

// ExtFrame reports whether the frame has a 29-bit extended
// identifier, rather than an 11-bit standard identifier.
func (f Flags) ExtFrame() bool {
	return f&ExtFrame != 0
}

// BusState is the fault confinement state of a CAN controller.
// The states are ordered: a larger value is a worse state.
type BusState uint8

const (
	ErrorActive  BusState = iota // normal operation
	ErrorWarning                 // an error counter reached 96
	ErrorPassive                 // an error counter reached 128
	BusOff                       // the transmit error counter exceeded 255
)

var busStateNames = [...]string{"error-active", "error-warning", "error-passive", "bus-off"}

func (s BusState) String() string {
	if int(s) < len(busStateNames) {
		return busStateNames[s]
	}
	return "BusState(" + strconv.Itoa(int(s)) + ")"
}

// Events are conditions reported by a status message.
type Events uint16

const (
	NoAck      Events = 1 << iota // a frame this node sent was not acknowledged
	BusError                      // bit, stuff, form or CRC error
	RxOverflow                    // received frames were lost
	TxTimeout                     // a frame could not be sent in time
	Restarted                     // the controller restarted after bus-off
)

var eventNames = [...]string{"no-ack", "bus-error", "rx-overflow", "tx-timeout", "restarted"}

func (e Events) String() string {
	var names []string
	for i, name := range eventNames {
		if e&(1<<i) != 0 {
			names = append(names, name)
		}
	}
	return strings.Join(names, ",")
}

// Status describes the state of a CAN controller,
// as reported by a status message.
type Status struct {
	State    BusState
	Events   Events // the conditions that caused the status message
	TxErr    uint8  // transmit error counter, if Counters
	RxErr    uint8  // receive error counter, if Counters
	Counters bool   // TxErr and RxErr are valid
}

// A Msg is a CAN frame, or a status message (see [Msg.Status]).
//
// Payloads of up to eight bytes are stored in the Msg itself,
// so copying such a Msg copies its payload. Storage for longer
// payloads is allocated on demand, reused by later SetData and Read
// calls, and shared by copies; use [Msg.Clone] for an independent copy.
type Msg struct {
	Id uint32 // The CAN message identifier
	Flags

	n      uint8   // payload length, if stored inline
	inline [8]byte // classic payload
	ext    []byte  // payload longer than eight bytes; storage reused

	isStatus bool
	status   Status

	Rx struct {
		Time Time // Timestamp
	}
}

// Data returns the payload. The slice is valid until the next
// SetData, Reset, or Read into m.
//
// A payload of up to eight bytes is stored in m itself, and the
// returned slice has capacity eight. It may be resliced, filled,
// and passed to SetData, which then only records the new length.
func (m *Msg) Data() []byte {
	if len(m.ext) > 0 {
		return m.ext
	}
	return m.inline[:m.n]
}

// SetData sets m's payload to the contents of b, making m a frame
// if it was a status message. It does not retain b.
// If b is m's own payload storage, as returned by Data and resliced,
// only the length is updated.
func (m *Msg) SetData(b []byte) {
	m.isStatus = false
	if len(b) <= len(m.inline) {
		if len(b) > 0 && &b[0] != &m.inline[0] {
			copy(m.inline[:], b)
		}
		m.n = uint8(len(b))
		m.ext = m.ext[:0]
		return
	}
	m.n = 0
	if cap(m.ext) > 0 && &b[0] == &m.ext[:1][0] {
		m.ext = m.ext[:len(b)] // b lies within ext's capacity
		return
	}
	m.ext = append(m.ext[:0], b...)
}

// Clone returns a copy of m that does not share payload storage with m.
func (m *Msg) Clone() Msg {
	c := *m
	c.ext = nil
	if len(m.ext) > 0 {
		c.ext = bytes.Clone(m.ext)
	}
	return c
}

// IsStatus reports whether m is a status message rather than a frame.
// A status message has no identifier, flags or payload.
func (m *Msg) IsStatus() bool {
	return m.isStatus
}

// Status returns the status a status message reports.
// It reports false if m is a frame.
func (m *Msg) Status() (Status, bool) {
	return m.status, m.isStatus
}

// SetStatus makes m a status message reporting st.
// It keeps m's payload storage for reuse.
func (m *Msg) SetStatus(st Status) {
	m.Reset()
	m.isStatus = true
	m.status = st
}

// Reset sets m back to the zero message. It keeps
// m's payload storage for reuse.
func (m *Msg) Reset() {
	ext := m.ext[:0]
	*m = Msg{ext: ext}
}

var ValidFDSizes = []int{12, 16, 20, 24, 32, 48, 64}

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
			m.Flags |= FDF
			expr = expr[1:]
			if expr == "" {
				return errors.New("CAN FD flags value missing")
			}
			u, err := strconv.ParseInt(expr[:1], 16, 8)
			if err != nil {
				return err
			}
			if u&1 != 0 {
				m.Flags |= BRS
			}
			if u&2 != 0 {
				m.Flags |= ESI
			}
			// Bit 4 (FDF) is implied by the separator.
			expr = expr[1:]
		}
		if expr == "R" {
			m.Flags |= RTRMsg
			return nil
		}
	}
	data, err := hex.DecodeString(dots.Replace(expr))
	if err != nil {
		return err
	}
	m.SetData(data)
	return nil
}

var dots = strings.NewReplacer(".", "")

func (m *Msg) Format() string {
	if st, ok := m.Status(); ok {
		return formatStatus(st)
	}
	var sb strings.Builder

	idFormat := "%03X:"
	if m.ExtFrame() {
		idFormat = "%08X:"
	}
	fmt.Fprintf(&sb, idFormat, m.Id)
	if m.Flags&RTRMsg != 0 {
		sb.WriteByte('R')
		return sb.String()
	}
	// As in can-utils' cansend, a doubled separator marks an FD frame,
	// followed by a digit with BRS (1) and ESI (2).
	data := m.Data()
	if m.Flags&(FDF|BRS) != 0 || len(data) > 8 {
		fd := 0
		if m.Flags&BRS != 0 {
			fd |= 1
		}
		if m.Flags&ESI != 0 {
			fd |= 2
		}
		fmt.Fprintf(&sb, ":%X", fd)
	}
	x := strings.ToUpper(hex.EncodeToString(data))
	sb.WriteString(x)
	return sb.String()
}

// formatStatus returns a line such as
// "status: error-passive no-ack (tx 128, rx 0)".
func formatStatus(st Status) string {
	s := "status: " + st.State.String()
	if st.Events != 0 {
		s += " " + st.Events.String()
	}
	if st.Counters {
		s += fmt.Sprintf(" (tx %d, rx %d)", st.TxErr, st.RxErr)
	}
	return s
}
