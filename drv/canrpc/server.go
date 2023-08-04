// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canrpc

import (
	"io"

	"github.com/knieriem/can"
)

type object struct {
	dev can.Device

	rch       rdChannels
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
	o.rch = channelizeReader(dev, nil)
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
	w.Data = m.Data()
	w.Flags = m.Flags
}

func (w *WireMsg) decode(m *can.Msg) {
	m.Id = w.ID
	m.SetData(w.Data)
	m.Flags = w.Flags
}

func (d *object) Read(nw int, r *[]WireMsg) (err error) {
	select {
	case in := <-d.rch.Data:
		n := len(in.Data)
		if n > nw {
			n = nw
		}
		//		fmt.Fprint(os.Stderr, "r")
		err = in.Err
		if err == nil {
			*r = make([]WireMsg, n)
			copy(*r, in.Data[:n])
		}
		d.rch.Req <- n
	case <-d.unblockch:
		//		fmt.Fprint(os.Stderr, "U")
		err = io.EOF
	}
	return
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

type data struct {
	Data []WireMsg
	Err  error
}

type rdChannels struct {
	Req  chan int
	Data chan data
}

func loop(r can.Device, buf []WireMsg, ch rdChannels) {
	var (
		d     data
		avail = buf[:0]
		n     int
	)
	mbuf := make([]can.Msg, 1)
	mbuf[0].SetData(make([]byte, 0, 64))

	for {
		if len(avail) == 0 {
			if nread, err := r.Read(mbuf); err == nil {
				avail = buf[:nread]
				(&buf[0]).encode(&mbuf[0])
			} else {
				d.Err = err
			}
		}
		d.Data = avail

		select {
		case ch.Data <- d:
			n = <-ch.Req
		case n = <-ch.Req:
		}
		if n == 0 {
			break
		}
		avail = avail[n:]
	}
	close(ch.Data)
}

// Create a wrapper around the Reader of a can.Device. A pair of
// channels is returned that can be used to request a number of messages,
// and to receive a []can.Msg containing the messages actually read.
func channelizeReader(r can.Device, buf []WireMsg) (ch rdChannels) {
	if buf == nil {
		buf = make([]WireMsg, 32)
	}
	ch.Req = make(chan int)
	ch.Data = make(chan data)

	go loop(r, buf, ch)

	return
}
