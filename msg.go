// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can

import "errors"

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
