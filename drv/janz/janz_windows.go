// Copyright 2014 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows,386

package janz

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
	"unsafe"

	"can"
	"can/drv"
	"can/drv/janz/pcan"
)

var d *driver

func init() {
	if dllPresent("jpcan.dll") && dllPresent("jpcangohelper.dll") {
		d = new(driver)
		can.RegisterDriver(d)
	}
}

func dllPresent(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

const (
	defaultBitrate = 0x001c
)

var builtinBitrates = map[can.Bitrate]uint16{
	1000000: 0x0014,
	800000:  0x0016,
	500000:  0x001C,
	250000:  0x011C,
	125000:  0x031C,
}

type driver struct {
}

func (*driver) Name() string {
	return "janz"
}

func scanOptions(list []interface{}) (bitrate uint16, term bool, err error) {
	bitrate = defaultBitrate
	for _, opt := range list {
		switch v := opt.(type) {
		case can.Bitrate:
			if b, ok := builtinBitrates[v]; ok {
				bitrate = b
			} else {
				err = errors.New("bitrate not supported")
				return
			}
		case can.Termination:
			term = bool(v)
		}
	}
	return
}

type dev struct {
	name can.Name
	fd   int
	term bool
}

func (*driver) Scan() (list []can.Name) {
	var di = make([]pcan.DeviceInfo, 16)

	n, err := pcan.USBDevices(di)
	if err != nil {
		return
	}

	for i := range di[:n] {
		name := di[i].Name()
		id := name
		if strings.HasPrefix(id, "canusb") {
			id = id[3:]
		}
		list = append(list, can.Name{
			ID:      id,
			Device:  name,
			Display: di[i].Desc(),
			Driver:  "janz",
		})
	}
	return
}

func (*driver) Open(devName string, options ...interface{}) (cd can.Device, err error) {
	var name can.Name

	defer wrapErr("open", &err)

	sc := d.Scan()
	if devName == "" {
		if len(sc) == 0 {
			err = errors.New("no channels available")
			return
		}
		name = sc[0]
	} else {
		for i := range sc {
			if sc[i].ID == devName {
				name = sc[i]
				goto found
			}
		}
		err = errors.New("not found")
		return
	}

found:
	bitrate, term, err := scanOptions(options)
	if err != nil {
		return
	}

	fd, err := pcan.Open(name.Device)
	if err != nil {
		return
	}
	d := new(dev)
	d.fd = fd
	d.name = name

	err = pcan.InitPool(fd, 1000)
	if err != nil {
		goto close
	}
	err = pcan.CreateHelper(fd)
	if err != nil {
		goto close
	}

	err = pcan.SetBTR(fd, (bitrate<<8)|(bitrate>>8))
	if err != nil {
		goto closeHelper
	}
	err = pcan.BusOn(fd)
	if err != nil {
		goto closeHelper
	}
	if term {
		err = pcan.ConfigTerm(fd, 1)
	} else {
		err = pcan.ConfigTerm(fd, 0)
	}
	if err != nil {
		return
	}
	d.term = term

	cd = d
	return

closeHelper:
	pcan.CloseHelper(fd)
close:
	pcan.Close(fd)
	return
}

func (d *dev) ID() string {
	return "janz:" + d.name.ID
}

func (d *dev) Read(buf []can.Msg) (n int, err error) {
	var b [32]uint8
	for {
		err = pcan.ReadMsg(d.fd, &b[0])
		if err != nil {
			return
		}
		ok := decode(&buf[n], b[:])
		if ok {
			n++
		}
		if n == len(buf) {
			break
		}
		if pcan.MsgAvail(d.fd) == 0 {
			break
		}
	}
	return
}

const (
	queueOverflow = -18
	evtErr        = 1
	evtBerr       = 16
	evtOverrun    = 2
)

func (d *dev) Write(msgs []can.Msg) (n int, err error) {
	for i := range msgs {
		err = d.WriteMsg(&msgs[i])
		if err != nil {
			break
		}
		n++
	}
	return
}

func (d *dev) WriteMsg(cm *can.Msg) (err error) {
	var m pcan.MsgData

	defer wrapErr("write", &err)

	if cm.IsStatus() {
		return
	}
	encode(&m, cm)

retry:
	rc, err := pcan.SendMsg(d.fd, &m)
	if err != nil {
		if rc == queueOverflow {
			// simulate blocking
			time.Sleep(100 * time.Millisecond)
			log.Println(err, "retry")
			goto retry
		}
	}

	return
}

var msgFlagsMap = drv.FlagsMap{
	{can.RTRMsg, 1 << 6},
	{can.ExtFrame, 1 << 7},
}

func decode(dst *can.Msg, b []byte) (ok bool) {
	switch b[0] {
	case 1:
		dst.Rx.Time = can.Now()
		dst.Flags = msgFlagsMap.Decode(int(b[1]))
		dst.Len = int(b[1]) & 0xF
		dst.Id = *(*uint32)(unsafe.Pointer(&b[4]))
		copy(dst.Data[:], b[8:16])
		ok = true
	case 2:
		dst.Flags = can.StatusMsg
		ok = true
		switch b[1] {
		case evtErr:
			dst.Flags |= can.ErrorPassive
		case evtBerr:
			dst.Flags |= can.BusOff
		case evtOverrun:
			dst.Flags |= can.DataOverrun
		default:
			ok = false
		}
	}
	return
}

func encode(dst *pcan.MsgData, src *can.Msg) {
	dst.ID = src.Id
	dst.Len = byte(src.Len)
	dst.ExtID = 0
	if (src.Flags & can.ExtFrame) != 0 {
		dst.ExtID = 1
	}
	dst.RTR = 0
	if (src.Flags & can.RTRMsg) != 0 {
		dst.RTR = 1
	}
	copy(dst.Data[:src.Len], src.Data[:src.Len])
}

func (d *dev) Close() (err error) {
	if d.term {
		pcan.ConfigTerm(d.fd, 0)
	}
	pcan.BusOff(d.fd)
	pcan.CloseHelper(d.fd)
	pcan.Close(d.fd)
	return
}

func (d *dev) Version() (v can.Version) {
	var lib, fw uint32

	err := pcan.Idvers(d.fd, &lib, &fw)
	if err != nil {
		return
	}
	vStr := func(u uint32) string {
		return fmt.Sprintf("%d.%d.%d", byte(u>>24), byte(u>>16), u&0xFFFF)
	}
	v.Device = vStr(fw)
	v.Api = vStr(lib)

	return
}

func (d *dev) Name() can.Name {
	return d.name
}

type wrappedError struct {
	error
	fnName string
}

func (e wrappedError) Error() string {
	return "janz: " + e.fnName + ": " + e.error.Error()
}

func wrapErr(fnName string, pErr *error) {
	err := *pErr
	if err != nil {
		*pErr = &wrappedError{err, fnName}
	}
}
