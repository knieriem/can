// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can

// Message Flags.
type Flags int

const (
	// message type
	ExtFrame Flags = 1 << iota
	RTRMsg
	StatusMsg

	// if StatusMsg is set:
	ErrorActive
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

// Definition of a CAN Message.
type Msg struct {
	Id uint32 // The CAN message identifier
	Flags
	Len  int
	Data [8]byte

	Tx struct {
		Delayµs int // The driver shall delay this message.
	}
	Rx struct {
		Time Time // Timestamp
	}
}
