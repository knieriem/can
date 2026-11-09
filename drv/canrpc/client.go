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
	"time"

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
	lost   atomic.Bool // the server did not reply to a ping in time

	infoOnce sync.Once
	info     can.DeviceInfo

	// WriteMsg queues messages; flush sends them with one Write call
	// at a time, which keeps their order. The server runs concurrent
	// calls in separate goroutines, so it would not.
	wmu      sync.Mutex
	flushed  sync.Cond // signaled when flush ends
	pending  []WireMsg
	flushing bool
	werr     error // error of an earlier Write call, reported once
}

// ErrConnLost is returned, wrapped, after the server stopped replying,
// for example because a tunnel to it broke.
var ErrConnLost = errors.New("connection lost")

// The interval and timeout of pings; variables, so that tests can
// shorten them.
var (
	pingInterval = 10 * time.Second
	pingTimeout  = 30 * time.Second
)

// watch pings the server regularly and closes the connection when a
// ping gets no reply in time, so that a broken link, which TCP through
// a tunnel may not notice, does not block Read forever. Any reply
// counts, even the error of an earlier server without Ping.
func (d *device) watch() {
	t := time.NewTicker(pingInterval)
	defer t.Stop()
	for range t.C {
		if d.closed.Load() {
			return
		}
		replied := make(chan struct{})
		go func() {
			d.call("Ping", 0, nil)
			close(replied)
		}()
		select {
		case <-replied:
		case <-time.After(pingTimeout):
			d.lost.Store(true)
			d.Closer.Close()
			return
		}
	}
}

// maxPending limits the messages queued by WriteMsg.
const maxPending = 1024

func newDevice(c Caller, path string) *device {
	d := &device{Caller: c, path: path}
	d.flushed.L = &d.wmu
	return d
}

func NewDevice(c Caller, name string) can.Device {
	return newDevice(c, name)
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

	// split the optional path part from addr
	path := ""
	if i := strings.Index(addr, "/"); i != -1 {
		addr, path = addr[:i], addr[i+1:]
	}

	cl, err := rpc.Dial("tcp", addr)
	if err != nil {
		return
	}
	d := newDevice(cl, path)
	d.Closer = cl
	go d.watch()
	cd = d
	return
}

// Info returns the remote device's information, with "rpc:" prefixed
// to the driver name, or only "rpc" from a server before v0.6.
func (d *device) Info() *can.DeviceInfo {
	d.infoOnce.Do(func() {
		var info can.DeviceInfo
		if d.call("Info", 0, &info) != nil {
			d.info = can.DeviceInfo{ID: "rpc", Driver: "rpc"}
			return
		}
		info.Driver = "rpc:" + info.Driver
		d.info = info
	})
	return &d.info
}

func (d *device) ID() string {
	return d.Info().ID
}

func (d *device) Read(buf []can.Msg) (n int, err error) {
	if d.closed.Load() {
		return 0, d.wrapErr("read", can.ErrClosed)
	}
	var r []WireMsg
	d.cl.Lock()
	err = d.call("Read", len(buf), &r)
	d.cl.Unlock()
	if err != nil {
		if d.closed.Load() {
			err = can.ErrClosed
		}
		return 0, d.wrapErr("read", err)
	}
	for i := range r {
		r[i].decode(&buf[i])
	}
	return len(r), nil
}

// WriteMsg queues m and returns. A Write call sends the queued messages
// in the background, so that writing is not limited to one message per
// round trip. An error of the remote device is returned by a later
// WriteMsg, or by Close, which waits until all messages are sent.
func (d *device) WriteMsg(m *can.Msg) error {
	if d.closed.Load() {
		return d.wrapErr("write", can.ErrClosed)
	}
	var w WireMsg
	w.encode(m)

	d.wmu.Lock()
	defer d.wmu.Unlock()
	if err := d.werr; err != nil {
		d.werr = nil
		return d.wrapErr("write", err)
	}
	if len(d.pending) >= maxPending {
		return d.wrapErr("write", can.ErrTxQueueFull)
	}
	d.pending = append(d.pending, w)
	if !d.flushing {
		d.flushing = true
		go d.flush()
	}
	return nil
}

// flush sends the queued messages, one Write call at a time,
// until none are left or a call fails.
func (d *device) flush() {
	d.wmu.Lock()
	defer d.wmu.Unlock()
	for len(d.pending) > 0 {
		batch := d.pending
		d.pending = nil
		d.wmu.Unlock()
		var n int
		err := d.call("Write", batch, &n)
		d.wmu.Lock()
		if err != nil {
			d.werr = err
			d.pending = nil // keep the order: drop what follows a failure
			break
		}
	}
	d.flushing = false
	d.flushed.Broadcast()
}

func (d *device) Close() error {
	if d.closed.Swap(true) {
		return d.wrapErr("close", can.ErrClosed)
	}

	// Wait until queued messages are sent.
	d.wmu.Lock()
	for d.flushing {
		d.flushed.Wait()
	}
	werr := d.werr
	d.wmu.Unlock()

	// Unblock a pending Read on the server side.
	err := d.call("Close", 0, nil)
	if err == nil {
		err = werr
	}

	if d.Closer != nil {
		// Wait until the last Read call returned before closing
		// the RPC connection.
		d.cl.Lock()
		d.Closer.Close()
		d.cl.Unlock()
	}
	if err != nil {
		return d.wrapErr("close", err)
	}
	return nil
}

// wrapErr adds context to err. As net/rpc transports server errors as
// strings only, it recognizes ErrClosed and ErrTxQueueFull by their text.
func (d *device) wrapErr(op string, err error) error {
	var se rpc.ServerError
	switch {
	case d.lost.Load() && !errors.Is(err, can.ErrClosed):
		err = ErrConnLost
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
