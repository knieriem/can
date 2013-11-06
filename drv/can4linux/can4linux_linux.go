// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,386

package can4linux

import (
	"io"
	"log"
	"os"
	"strconv"
	"syscall"

	"can"
)

type dev struct {
	fd     int
	rx, tx struct {
		epoll *pollster
		buf   []msg
	}
	cl io.Closer
	can.Unversioned
}

func (d *dev) Close() (err error) {
	d.rx.epoll.Close()
	d.tx.epoll.Close()
	return d.cl.Close()
}

func init() {
	f, err := os.Open("/proc/sys/Can/Chipset")
	if err != nil {
		return
	}
	f.Close()
	can.RegisterDriver(new(driver))
}

type driver struct {
}

func (*driver) Name() string {
	return "can4linux"
}

func (*driver) Scan() (list []string) {
	for i := 0; i < 4; i++ {
		n := strconv.Itoa(i)
		if fi, err := os.Stat("/dev/can" + n); err == nil {
			if fi.Mode()&os.ModeDevice != 0 {
				list = append(list, n)
			}
		}
	}
	return
}

func (drv *driver) Open(devName string) (cd can.Device, err error) {
	d := new(dev)

	if devName == "" {
		devName = "0"
		if list := drv.Scan(); len(list) != 0 {
			devName = list[0]
		}
	}
	f, err := os.OpenFile("/dev/can"+devName, os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		return
	}
	d.fd = int(f.Fd())

	if d.rx.epoll, err = newpollster(); err != nil {
		return
	}
	if _, err = d.rx.epoll.AddFD(d.fd, 'r', true); err != nil {
		return
	}

	if d.tx.epoll, err = newpollster(); err != nil {
		return
	}
	if _, err = d.tx.epoll.AddFD(d.fd, 'w', true); err != nil {
		return
	}
	d.tx.buf = make([]msg, 1)

	d.cl = f
	cd = d
	return
}

func (d *dev) Read(buf []can.Msg) (n int, err error) {
	r := &d.rx
	if len(r.buf) < len(buf) {
		r.buf = make([]msg, len(buf))
	}
	fd, _, err := r.epoll.WaitFD(0)
	if err != nil {
		return
	}
	if n, err = readMsg(fd, r.buf[:len(buf)]); err != nil {
		return
	}

	for i := range r.buf[:n] {
		r.buf[i].decode(&buf[i])
	}
	return
}

func (d *dev) Write(buf []can.Msg) (n int, err error) {
	w := &d.tx
	if len(w.buf) < len(buf) {
		w.buf = make([]msg, len(buf))
	}
	fd, _, err := w.epoll.WaitFD(0)
	if err != nil {
		return
	}
	for i := range buf {
		if buf[i].IsStatus() {
			continue
		}
		w.buf[n].encode(&buf[i])
		n++
	}
	n, err = writeMsg(fd, w.buf[:n])
	return
}

func (d *dev) WriteMsg(m *can.Msg) (err error) {
	if m.IsStatus() {
		return
	}
	w := &d.tx
	w.buf[0].encode(m)
	_, _, err = w.epoll.WaitFD(0)
	_, err = writeMsg(d.fd, w.buf[:1])
	if err != nil {
		return
	}
	return
}

const (
	// ioctl actions
	cmdAction = iota
	confAction
	_
	_
	_
	statusAction
)

const (
	// commands
	cmdRun   = 1
	cmdRtop  = 2
	cmdReset = 3
)

//sys	ioctlCmd(fd int, action int, p *ioctlCmdArg) (err error) = SYS_IOCTL

const (
	// configuration items
	confAcceptanceMaskAndCode = iota
	confAcceptanceMask
	confAcceptanceCode
	confTiming
	confOMode
	confFilter
	confFilterEnable
	confFilterDisable
)

//sys	ioctlConf(fd int, action int, p *ioctlConfArg) (err error) = SYS_IOCTL
//sys	ioctlStatus(fd int, action int, p *ioctlStatusArg) (err error) = SYS_IOCTL

//sys readMsg(fd int, p []msg) (n int, err error) = SYS_READ
//sys writeMsg(fd int, p []msg) (n int, err error) = SYS_WRITE

func (m *msg) decode(cm *can.Msg) {
	f := int(m.Flags)
	cm.Flags = MsgFlagsMap.Decode(f)
	if m.Id == 0xFFFFFFFF && f&ErrMask != 0 {
		cm.Id = 0
		cm.Flags |= ErrFlagsMap.Decode(f)
	} else {
		cm.Id = m.Id
	}
	cm.Len = int(m.Length)
	copy(cm.Data[:], m.Data[:])

	cm.Tx.Delayµs = 0
	cm.Rx.Time = can.UnixTimevals(m.Tstamp.Sec, m.Tstamp.Usec)
}

func (m *msg) encode(cm *can.Msg) {
	m.Id = cm.Id

	m.Flags = 0
	if cm.Flags&can.RTRMsg != 0 {
		m.Flags |= remoteTransmissionReq
	}
	if cm.Flags&can.ExtFrame != 0 {
		m.Flags |= extFrameFormat
	}

	m.Length = int16(cm.Len)
	copy(m.Data[:], cm.Data[:])

	m.Tstamp.Usec = int32(cm.Tx.Delayµs)
	m.Tstamp.Sec = 0
}

func (d *dev) status() {
	var st ioctlStatusArg

	err := ioctlStatus(d.fd, statusAction, &st)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("%#v ", st)
}

var _zero uintptr
