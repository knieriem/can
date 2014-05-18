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
// 	rpc:192.168.1.2:6000
//
// An optional name can be appended, prefixed by a slash, if the
// server is exporting multiple CAN devices:
//
//	rpc:192.168.1.2:6000/device2
//
// The name must, of course, match the one configured in the server.
package canrpc

import (
	"io"
	"net/rpc"
	"strings"
	"sync"

	"can"
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
	path string
	cl   sync.Mutex
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

func (*driver) Scan() (list []can.Name) {
	return
}

func (*driver) Open(addr string, _ ...interface{}) (cd can.Device, err error) {

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
	return "pcan"
}

func (d *device) Read(buf []can.Msg) (n int, err error) {
	var r []can.Msg
	d.cl.Lock()
	err = d.call("Read", len(buf), &r)
	d.cl.Unlock()
	if err == nil {
		copy(buf, r)
		n = len(r)
	}
	return
}

func (d *device) Write(buf []can.Msg) (n int, err error) {
	err = d.call("Write", buf, &n)
	return
}
func (d *device) WriteMsg(m *can.Msg) (err error) {
	err = d.call("WriteMsg", m, nil)
	return
}

func (d *device) Close() (err error) {
	err = d.call("Close", 0, nil)

	if d.Closer == nil {
		return
	}
	// before closing the RPC connection, wait until the last Read
	// call returned, which should have been interrupted by the
	// Close call above.
	d.cl.Lock()
	d.Closer.Close()
	d.cl.Unlock()
	return
}

func (d *device) call(fnName string, arg, reply interface{}) error {
	name := "Can"
	if d.path != "" {
		name += "-" + d.path
	}
	return d.Call(name+"."+fnName, arg, reply)
}
