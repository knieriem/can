// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pcan

import (
	"encoding/binary"
	"errors"
	"os"
	"runtime"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv"
	"github.com/knieriem/can/drv/pcan/internal/api"
)

func driverPresent() bool {
	return true
}

type bus struct {
	name     string
	channels []channel
}

type channel struct {
	minor  int
	netdev string
}

func (b *bus) getFirstAvail() (i int) {
	if b.nChan() == 0 {
		i = -1
	}
	return
}

func (b *bus) canAutoDetect() bool {
	return true
}

func (b *bus) sysName(minor int) string {
	return "/dev/pcan" + b.name + strconv.Itoa(minor)
}

func (*driver) Scan() (list []can.DeviceInfo) {

	buses, err := parseProcfile()
	if err != nil {
		return
	}

	for _, b := range buses {
		for i, ch := range b.channels {
			list = append(list, can.DeviceInfo{
				ID:     b.name + strconv.Itoa(i+1),
				Driver: "pcan",
				Device: b.sysName(ch.minor),
			})
		}
	}
	return
}

// dev accesses a pcan character device. The file is registered with
// Go's runtime poller (os.OpenFile does this for pollable descriptors),
// and all ioctls are issued through its syscall.RawConn, so that
//   - a blocked Read parks the goroutine in the poller, not an OS thread,
//   - Close unblocks a pending Read, which returns can.ErrClosed,
//   - the descriptor is never closed while an ioctl is using it.
//
// Note: never call d.file.Fd(), it would switch the file back to
// blocking mode and detach it from the poller.
type dev struct {
	file    *os.File
	rc      syscall.RawConn
	info    can.DeviceInfo
	receive struct {
		status api.Status
		t0     can.Time
		t0val  int64
	}
}

func (*driver) Open(devName string, conf *can.Config) (cd can.Device, err error) {
	defer wrapErr("open", &err)

	hwList, err := parseProcfile()
	if err != nil {
		return
	}

	bus, iDev, err := hwList.lookupName(devName)
	if err != nil {
		return
	}
	ch := bus.channels[iDev]
	if ch.netdev != "" {
		err = errors.New("driver configured in netdev mode (need chardev)")
		return
	}

	bitrate, err := timingConf(conf)
	if err != nil {
		return nil, err
	}

	sysName := bus.sysName(ch.minor)
	f, err := os.OpenFile(sysName, os.O_RDWR, 0)
	if err != nil {
		return
	}
	d := new(dev)
	d.file = f
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	if d.rc, err = f.SyscallConn(); err != nil {
		return nil, err
	}
	d.info = can.DeviceInfo{
		ID:     bus.name + strconv.Itoa(iDev+1),
		Device: sysName,
		Driver: "pcan",
	}
	d.setupInfo()

	var i api.Init
	i.WBTR0BTR1 = bitrate
	i.UcCANMsgType = api.MsgExtended
	err = d.control(func(h api.Fd) error { return h.Init(&i) })
	if err != nil {
		if runtime.GOARCH == "386" && err == syscall.EINVAL {
			err = errors.New("32-bit program / 64-bit driver mismatch")
		}
		return nil, err
	}

	err = d.control(func(h api.Fd) error { return h.SetMsgFilter(nil) })
	if err != nil {
		return nil, err
	}
	return d, nil
}

// control runs fn with the raw descriptor, without waiting for readiness.
func (d *dev) control(fn func(api.Fd) error) error {
	var err error
	if cerr := d.rc.Control(func(fd uintptr) { err = fn(api.Fd(fd)) }); cerr != nil {
		return cerr
	}
	return err
}

var errNoMsg = errors.New("no message pending")

// pollReady reports, without blocking, whether fd is ready for events.
func pollReady(fd uintptr, events int16) bool {
	pfd := []unix.PollFd{{Fd: int32(fd), Events: events}}
	for {
		n, err := unix.Poll(pfd, 0)
		if err == unix.EINTR {
			continue
		}
		return err == nil && n > 0 && pfd[0].Revents&events != 0
	}
}

// readMsg reads one message from the driver queue. If wait is true,
// it blocks in the runtime poller until a message is available;
// otherwise it returns errNoMsg if the queue is empty.
//
// Readiness is checked with a zero-timeout poll before the ioctl,
// so this does not depend on whether the pcan driver honors
// O_NONBLOCK for its READ_MSG ioctl.
func (d *dev) readMsg(m *api.RMsg, wait bool) error {
	var err error
	cerr := d.rc.Read(func(fd uintptr) bool {
		if !pollReady(fd, unix.POLLIN) {
			if wait {
				return false // park until the poller reports readability
			}
			err = errNoMsg
			return true
		}
		err = api.Fd(fd).ReadMsg(m)
		return !(wait && errors.Is(err, unix.EAGAIN))
	})
	if cerr != nil {
		return cerr
	}
	return err
}

func (d *dev) Read(buf []can.Msg) (n int, err error) {
	var m api.RMsg

	defer wrapErr("read", &err)

	prevSt := d.receive.status
	defer func() {
		d.receive.status = prevSt
	}()

	for n < len(buf) {
		err1 := d.readMsg(&m, n == 0)
		if err1 != nil {
			if err1 == errNoMsg || n > 0 {
				return n, nil
			}
			return 0, closedErr(err1)
		}

		µs := int64(m.DwTime)*1000 + int64(m.WUsec)
		st := d.decode(&buf[n], &m.Msg, µs)

		switch {
		case st == api.OK:
			prevSt = st
			n++
		case st.Test(api.ErrANYBUSERR):
			st &= api.ErrANYBUSERR
			if st != prevSt {
				prevSt = st
				n++
				return
			}
		}
	}
	return
}

var msgFlagsMap = drv.FlagsMap{
	{can.RTRMsg, api.MsgRtr},
	{can.ExtFrame, api.MsgExtended},
}

func (d *dev) decode(dst *can.Msg, m *api.Msg, µs int64) (st api.Status) {

	if d.receive.t0 == 0 {
		d.receive.t0 = can.Now()
		d.receive.t0val = µs
	}
	dst.Rx.Time = d.receive.t0 + can.Time(µs-d.receive.t0val)

	if m.MSGTYPE&api.MsgStatus != 0 {
		st = api.Status(binary.BigEndian.Uint32(m.DATA[0:4]))
		dst.Flags = errFlagsMap.Decode(int(st))
		dst.Flags |= can.StatusMsg
		dst.Id = 0
		dst.SetData(nil)
		return
	}
	dst.Id = m.ID
	dst.Flags = msgFlagsMap.Decode(int(m.MSGTYPE))
	dst.SetData(m.DATA[:min(int(m.LEN), len(m.DATA))])
	return
}

func (d *dev) WriteMsg(cm *can.Msg) (err error) {
	var m api.Msg

	defer wrapErr("write", &err)

	if cm.IsStatus() {
		return
	}
	if len(cm.Data()) > 8 || cm.Flags&(can.FDF|can.BRS) != 0 {
		return can.ErrInvalidMsgLen
	}
	encode(&m, cm)

	// If the driver's TX queue is full, the ioctl returns EAGAIN, as
	// the file is in non-blocking mode. (A driver ignoring O_NONBLOCK
	// would block here instead.)
	err = d.control(func(h api.Fd) error { return h.WriteMsg(&m) })
	if errors.Is(err, unix.EAGAIN) {
		err = can.ErrTxQueueFull
	}
	return closedErr(err)
}

func (d *dev) Close() (err error) {
	err = closedErr(d.file.Close())
	wrapErr("close", &err)
	return
}

// closedErr translates the error of an operation on a closed file
// into can.ErrClosed.
func closedErr(err error) error {
	if errors.Is(err, os.ErrClosed) {
		return can.ErrClosed
	}
	return err
}

func (d *dev) setupInfo() {
	var diag api.Diag

	err := d.control(func(h api.Fd) error { return h.Diag(&diag) })
	if err == nil {
		d.info.SystemDriver = diag.Version()
	}
}

func (d *dev) Info() *can.DeviceInfo {
	return &d.info
}
