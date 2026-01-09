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
	data       []byte
	stdPayload [8]byte

	Tx struct {
		Delayµs int // The driver shall delay this message.
	}
	Rx struct {
		Time Time // Timestamp
	}
}

// Data returns the current Payload of the message.
// If no payload buffer has been set by calling SetData before,
// Data returns a byte slice with the standard payload length 8.
func (m *Msg) Data() []byte {
	if m.data == nil {
		return m.stdPayload[:0]
	}
	return m.data
}

// SetData updates the payload of the message.
func (m *Msg) SetData(b []byte) {
	m.data = b
}

// Reset sets the message back to the initial state.
func (m *Msg) Reset() {
	m.Id = 0
	m.Flags = 0
	m.data = m.data[:0]
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
