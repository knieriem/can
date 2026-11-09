// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canrpc

import (
	"bytes"
	"errors"
	"io"
	"time"

	"github.com/knieriem/can"
)

type object struct {
	dev can.Device

	rch       <-chan WireMsg // messages read from dev; closed after a read error
	err       error          // the read error; valid once rch is closed
	unblockch chan bool
}

// Satisfied by any object that has a RegisterName method,
// like *rpc.Server.
type Registrar interface {
	RegisterName(string, interface{}) error
}

// Register a can.Device with an RPC server.
func Register(r Registrar, dev can.Device, name string) error {
	o := new(object)
	o.dev = dev
	o.unblockch = make(chan bool, 1)
	ch := make(chan WireMsg)
	o.rch = ch
	go o.readLoop(ch)
	if name != "" {
		name = "Can-" + name
	} else {
		name = "Can"
	}

	return r.RegisterName(name, o)
}

type WireMsg struct {
	ID uint32
	can.Flags
	Data   []byte
	Status *can.Status // set for status messages; earlier versions ignore it
}

// Status flags of versions before v0.6. Status messages still carry
// them on the wire, so that earlier clients recognize them, and a
// client recognizes status messages from an earlier server by them.
const (
	wireStatusMsg    can.Flags = 1 << 2
	wireMissingAck   can.Flags = 1 << 5
	wireErrorActive  can.Flags = 1 << 6
	wireErrorWarning can.Flags = 1 << 7
	wireErrorPassive can.Flags = 1 << 8
	wireBusOff       can.Flags = 1 << 9
	wireDataOverrun  can.Flags = 1 << 10
	wireRxOverflow   can.Flags = 1 << 11

	wireStatusBits = wireStatusMsg | wireMissingAck | wireErrorActive |
		wireErrorWarning | wireErrorPassive | wireBusOff |
		wireDataOverrun | wireRxOverflow
)

func (w *WireMsg) encode(m *can.Msg) {
	if st, ok := m.Status(); ok {
		*w = WireMsg{Flags: wireStatusMsg | wireFlags(st), Status: &st}
		return
	}
	w.ID = m.Id
	// Copy: m's storage is reused by the next Read,
	// while net/rpc may still be encoding w.
	w.Data = bytes.Clone(m.Data())
	w.Flags = m.Flags
	w.Status = nil
}

func (w *WireMsg) decode(m *can.Msg) {
	switch {
	case w.Status != nil:
		m.SetStatus(*w.Status)
	case w.Flags&wireStatusMsg != 0:
		m.SetStatus(wireStatus(w.Flags)) // from an earlier server
	default:
		m.Id = w.ID
		m.SetData(w.Data)
		m.Flags = w.Flags &^ wireStatusBits
	}
}

// wireFlags returns the status flags of earlier versions for st.
func wireFlags(st can.Status) can.Flags {
	f := [...]can.Flags{
		can.ErrorActive:  wireErrorActive,
		can.ErrorWarning: wireErrorWarning,
		can.ErrorPassive: wireErrorPassive,
		can.BusOff:       wireBusOff,
	}[st.State]
	if st.Events&can.NoAck != 0 {
		f |= wireMissingAck
	}
	if st.Events&can.RxOverflow != 0 {
		f |= wireRxOverflow
	}
	return f
}

// wireStatus translates the status flags of an earlier server.
func wireStatus(f can.Flags) can.Status {
	var st can.Status
	switch {
	case f&wireBusOff != 0:
		st.State = can.BusOff
	case f&wireErrorPassive != 0:
		st.State = can.ErrorPassive
	case f&wireErrorWarning != 0:
		st.State = can.ErrorWarning
	}
	if f&wireMissingAck != 0 {
		st.Events |= can.NoAck
	}
	if f&(wireRxOverflow|wireDataOverrun) != 0 {
		st.Events |= can.RxOverflow
	}
	return st
}

// Read returns at least one and at most nw messages. After the device
// failed, it returns the device's read error, also on later calls.
func (o *object) Read(nw int, r *[]WireMsg) error {
	if nw <= 0 {
		return nil
	}
	select {
	case w, ok := <-o.rch:
		if !ok {
			return o.err
		}
		*r = append((*r)[:0], w)
	case <-o.unblockch:
		return io.EOF
	}
	for len(*r) < nw {
		select {
		case w, ok := <-o.rch:
			if !ok {
				return nil // report o.err with the next call
			}
			*r = append(*r, w)
		default:
			return nil
		}
	}
	return nil
}

// Write sends the messages of a batch in order, and stops at the first
// error. Clients send all messages with Write.
func (o *object) Write(w []WireMsg, n *int) error {
	var m can.Msg
	for i := range w {
		w[i].decode(&m)
		if err := o.writeMsg(&m); err != nil {
			*n = i
			return err
		}
	}
	*n = len(w)
	return nil
}

// writeMsg waits out a full transmit queue for up to a second, as the
// client cannot retry a single message from the middle of a batch.
func (o *object) writeMsg(m *can.Msg) error {
	deadline := time.Now().Add(time.Second)
	for {
		err := o.dev.WriteMsg(m)
		if !errors.Is(err, can.ErrTxQueueFull) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(time.Millisecond)
	}
}

// WriteMsg serves clients of earlier versions,
// which sent one message per call.
func (o *object) WriteMsg(w *WireMsg, _ *int) (err error) {
	var m can.Msg

	w.decode(&m)
	err = o.dev.WriteMsg(&m)
	return
}

func (o *object) Close(_ int, _ *int) (err error) {
	select {
	case o.unblockch <- true:
	default:
	}
	return
}

// readLoop reads messages from the device and hands them to Read
// calls. After a read error, it records the error and closes ch.
func (o *object) readLoop(ch chan<- WireMsg) {
	defer close(ch)
	buf := make([]can.Msg, 16)
	for {
		n, err := o.dev.Read(buf)
		for i := range buf[:n] {
			var w WireMsg
			w.encode(&buf[i])
			ch <- w
		}
		if err != nil {
			o.err = err
			return
		}
	}
}
