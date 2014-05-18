// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pcan

import (
	"errors"
	"io"
	"os"
	"strconv"

	"can"
	"can/drv/pcan/api"
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

func (*driver) Scan() (list []string) {

	buses, err := parseProcfile()
	if err != nil {
		return
	}

	for _, b := range buses {
		for i := range b.channels {
			list = append(list, b.name+strconv.Itoa(i+1))
		}
	}
	return
}

type dev struct {
	file    io.Closer
	h       api.Fd
	receive struct {
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

	f, err := os.OpenFile("/dev/pcan"+strconv.Itoa(bus.channels[iDev].minor), os.O_RDWR, 0)
	if err != nil {
		return
	}
	d := new(dev)
	d.file = f
	d.h = api.Fd(f.Fd())

	bitrate, err := scanOptions(options)
	if err != nil {
		return
	}

	var i api.Init
	i.WBTR0BTR1 = bitrate
	i.UcCANMsgType = api.MsgExtended
	err = d.h.Init(&i)
	if err != nil {
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
	err = d.file.Close()
	wrapErr("close", &err)
	return
}

func (d *dev) DriverVersion() (ver string) {
	var diag api.Diag

	err := d.h.Diag(&diag)
	if err == nil {
		ver = diag.Version()
	}
	return
}
