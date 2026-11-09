// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package canrpc implements net/rpc client and server objects.
// It is possible to register a can.Device with a server, and to speak
// to the server via can.Device on the client side.
//
// The package automatically registers a driver with the can package
// drivers list under the name "rpc". If, for example, a server is
// running on host 192.168.1.2, port 6000, the address provided to
// to the can.Open call would be written as:
//
//	rpc:192.168.1.2:6000
//
// An optional name can be appended, prefixed by a slash, if the
// server is exporting multiple CAN devices:
//
//	rpc:192.168.1.2:6000/device2
//
// The name must, of course, match the one configured in the server.
package canrpc

import (
	"errors"
	"fmt"
	"io"
	"net/rpc"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/knieriem/can"
)

// Satisfied by any object that has a Call method as described,
// like *rpc.Client.
type Caller interface {
	Call(funcName string, arg, reply interface{}) error
}

// The methods of a device object issue calls to an RPC server. Because
// a device object satisfies the can.Device interface, it can be used
// transparently as a can.Device.
type device struct {
	Caller
	io.Closer
	path   string
	cl     sync.Mutex
	closed atomic.Bool
	can.Unversioned
}

func NewDevice(c Caller, name string) can.Device {
	d := new(device)
	d.Caller = c
	d.path = name
	return d
}

func init() {
	can.RegisterDriver(new(driver))
}

type driver struct{}

func (*driver) Name() string {
	return "rpc"
}

func (*driver) Scan() (list []can.DeviceInfo) {
	return
}

func (*driver) Open(addr string, _ *can.Config) (cd can.Device, err error) {

	d := new(device)

	// split the optional path part from addr
	if i := strings.Index(addr, "/"); i != -1 {
		addr, d.path = addr[:i], addr[i+1:]
	}

	cl, err := rpc.Dial("tcp", addr)
	if err != nil {
		return
	}
	d.Caller = cl
	d.Closer = cl
	cd = d
	return
}

func (d *device) ID() string {
	return "rpc"
}

func (d *device) Read(buf []can.Msg) (n int, err error) {
	if d.closed.Load() {
		return 0, wrapErr("read", can.ErrClosed)
	}
	var r []WireMsg
	d.cl.Lock()
	err = d.call("Read", len(buf), &r)
	d.cl.Unlock()
	if err != nil {
		if d.closed.Load() {
			err = can.ErrClosed
		}
		return 0, wrapErr("read", err)
	}
	for i := range r {
		r[i].decode(&buf[i])
	}
	return len(r), nil
}

func (d *device) WriteMsg(m *can.Msg) error {
	if d.closed.Load() {
		return wrapErr("write", can.ErrClosed)
	}
	var w WireMsg
	w.encode(m)
	if err := d.call("WriteMsg", w, nil); err != nil {
		return wrapErr("write", err)
	}
	return nil
}

func (d *device) Close() error {
	if d.closed.Swap(true) {
		return wrapErr("close", can.ErrClosed)
	}
	// Unblock a pending Read on the server side.
	err := d.call("Close", 0, nil)

	if d.Closer != nil {
		// Wait until the last Read call returned before closing
		// the RPC connection.
		d.cl.Lock()
		d.Closer.Close()
		d.cl.Unlock()
	}
	if err != nil {
		return wrapErr("close", err)
	}
	return nil
}

// wrapErr adds context to err. As net/rpc transports server errors as
// strings only, it recognizes ErrClosed and ErrTxQueueFull by their text.
func wrapErr(op string, err error) error {
	var se rpc.ServerError
	switch {
	case errors.Is(err, rpc.ErrShutdown):
		err = can.ErrClosed
	case errors.As(err, &se):
		if s := string(se); strings.Contains(s, can.ErrClosed.Error()) {
			err = can.ErrClosed
		} else if strings.Contains(s, can.ErrTxQueueFull.Error()) {
			err = can.ErrTxQueueFull
		}
	}
	return fmt.Errorf("canrpc: %s: %w", op, err)
}

func (d *device) call(fnName string, arg, reply interface{}) error {
	name := "Can"
	if d.path != "" {
		name += "-" + d.path
	}
	return d.Call(name+"."+fnName, arg, reply)
}
