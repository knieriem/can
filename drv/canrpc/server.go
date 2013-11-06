// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package canrpc

import (
	"io"

	"can"
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

func (d *object) Read(nw int, r *[]can.Msg) (err error) {
	select {
	case in := <-d.rch.Data:
		n := len(in.Data)
		if n > nw {
			n = nw
		}
		//		fmt.Fprint(os.Stderr, "r")
		err = in.Err
		if err == nil {
			*r = make([]can.Msg, n)
			copy(*r, in.Data[:n])
		}
		d.rch.Req <- n
	case <-d.unblockch:
		//		fmt.Fprint(os.Stderr, "U")
		err = io.EOF
	}
	return
}

func (o *object) Write(buf []can.Msg, n *int) (err error) {
	//	fmt.Fprint(os.Stderr, "w")
	*n, err = o.dev.Write(buf)
	return
}

func (o *object) WriteMsg(m *can.Msg, _ *int) (err error) {
	//	fmt.Fprint(os.Stderr, "w")
	err = o.dev.WriteMsg(m)
	return
}

func (o *object) Close(_ int, _ *int) (err error) {
	o.unblockch <- true
	return
}

type data struct {
	Data []can.Msg
	Err  error
}

type rdChannels struct {
	Req  chan int
	Data chan data
}

func loop(r can.Device, buf []can.Msg, ch rdChannels) {
	var (
		d     data
		avail = buf[:0]
		n     int
	)

	for {
		if len(avail) == 0 {
			d.Data = avail
			if nread, err := r.Read(buf); err == nil {
				avail = buf[:nread]
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
func channelizeReader(r can.Device, buf []can.Msg) (ch rdChannels) {
	if buf == nil {
		buf = make([]can.Msg, 4096)
	}
	ch.Req = make(chan int)
	ch.Data = make(chan data)

	go loop(r, buf, ch)

	return
}
