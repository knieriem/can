// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canrpc

import (
	"bytes"
	"io"

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
	Data []byte
}

func (w *WireMsg) encode(m *can.Msg) {
	w.ID = m.Id
	// Copy: m's storage is reused by the next Read,
	// while net/rpc may still be encoding w.
	w.Data = bytes.Clone(m.Data())
	w.Flags = m.Flags
}

func (w *WireMsg) decode(m *can.Msg) {
	m.Id = w.ID
	m.SetData(w.Data)
	m.Flags = w.Flags
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

func (o *object) Write(w []WireMsg, n *int) (err error) {
	//	fmt.Fprint(os.Stderr, "w")
	m := make([]can.Msg, len(w))
	for i := range w {
		w[i].decode(&m[i])
	}
	*n, err = o.dev.Write(m)
	return
}

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
