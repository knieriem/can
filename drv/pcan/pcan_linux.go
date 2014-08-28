// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pcan

import (
	"errors"
	"io"
	"os"
	"runtime"
	"strconv"
	"syscall"

	"can"
	"can/drv/pcan/api"
	"github.com/knieriem/g/syscall/epoll"
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

func (*driver) Scan() (list []can.Name) {

	buses, err := parseProcfile()
	if err != nil {
		return
	}

	for _, b := range buses {
		for i, ch := range b.channels {
			list = append(list, can.Name{
				ID:     b.name + strconv.Itoa(i+1),
				Driver: "pcan",
				Device: "/dev/pcan" + strconv.Itoa(ch.minor),
			})
		}
	}
	return
}

type dev struct {
	file    io.Closer
	h       api.Fd
	name    can.Name
	receive struct {
		epoll  *epoll.Pollster
		status api.Status
		t0     can.Time
		t0val  int64
	}
}

func (*driver) Open(devName string, options ...interface{}) (cd can.Device, err error) {
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

	sysName := "/dev/pcan" + strconv.Itoa(ch.minor)
	f, err := os.OpenFile(sysName, os.O_RDWR, 0)
	if err != nil {
		return
	}
	d := new(dev)
	d.file = f
	d.h = api.Fd(f.Fd())
	d.name = can.Name{
		ID:     bus.name + strconv.Itoa(iDev+1),
		Device: sysName,
		Driver: "pcan",
	}

	bitrate, err := scanOptions(options)
	if err != nil {
		return
	}

	var i api.Init
	i.WBTR0BTR1 = bitrate
	i.UcCANMsgType = api.MsgExtended
	err = d.h.Init(&i)
	if err != nil {
		if runtime.GOARCH == "386" && err == syscall.EINVAL {
			err = errors.New("32-bit program / 64-bit driver mismatch")
		}
		return
	}

	if d.receive.epoll, err = epoll.NewPollster(); err != nil {
		return
	}
	if d.receive.epoll.AddFD(int(d.h), 'r', true); err != nil {
		return
	}
	d.h.SetMsgFilter(nil)

	cd = d

	return
}

func (d *dev) Read(buf []can.Msg) (n int, err error) {
	var m api.RMsg

	defer wrapErr("read", &err)

	prevSt := d.receive.status
	defer func() {
		d.receive.status = prevSt
	}()

	for n < len(buf) {
		if n > 0 && d.h.Status().Test(api.ErrQRCVEMPTY) {
			return
		}
		_, mode, err1 := d.receive.epoll.WaitFD(1e9)
		if err1 != nil {
			err = err1
			return
		}
		if mode == 0 {
			// WaitFD timeout
			continue
		}
		err = d.h.ReadMsg(&m)
		if err != nil {
			break
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

func (d *dev) WriteMsg(cm *can.Msg) (err error) {
	var m api.Msg

	defer wrapErr("write", &err)

	if cm.IsStatus() {
		return
	}
	encode(&m, cm)

	err = d.h.WriteMsg(&m)

	return
}

func (d *dev) Close() (err error) {
	d.receive.epoll.Close()
	err = d.file.Close()
	wrapErr("close", &err)
	return
}

func (d *dev) Version() (v can.Version) {
	var diag api.Diag

	err := d.h.Diag(&diag)
	if err == nil {
		v.Driver = diag.Version()
	}
	return
}

func (d *dev) Name() can.Name {
	return d.name
}
