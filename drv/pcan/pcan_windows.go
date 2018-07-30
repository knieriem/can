// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pcan

import (
	"errors"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/pcan/api"
	win "github.com/knieriem/g/syscall"
)

func driverPresent() bool {
	_, err := exec.LookPath("PCANBasic.dll")
	return err == nil
}

type bus struct {
	name     string
	pnp      bool
	channels []api.Handle
}

func (b *bus) getFirstAvail() (i int) {
	for i, ch := range b.channels {
		if ch.Available() {
			return i
		}
	}
	return -1
}

func (b *bus) canAutoDetect() bool {
	return b.pnp
}

const (
	pnp   = true
	nopnp = false
)

var hwList = busList{
	{
		"usb", pnp, []api.Handle{
			api.USBBUS1,
			api.USBBUS2,
			api.USBBUS3,
			api.USBBUS4,
			api.USBBUS5,
			api.USBBUS6,
			api.USBBUS7,
			api.USBBUS8,
		},
	}, {
		"pci", pnp, []api.Handle{
			api.PCIBUS1,
			api.PCIBUS2,
			api.PCIBUS3,
			api.PCIBUS4,
			api.PCIBUS5,
			api.PCIBUS6,
			api.PCIBUS7,
			api.PCIBUS8,
		},
	}, {
		"pcc", pnp, []api.Handle{
			api.PCCBUS1,
			api.PCCBUS2,
		},
	}, {
		"dng", nopnp, []api.Handle{
			api.DNGBUS1,
		},
	}, {
		"isa", nopnp, []api.Handle{
			api.ISABUS1,
			api.ISABUS2,
			api.ISABUS3,
			api.ISABUS4,
			api.ISABUS5,
			api.ISABUS6,
			api.ISABUS7,
			api.ISABUS8,
		},
	},
}

type dev struct {
	name    can.Name
	h       api.Handle
	bus     *bus
	receive struct {
		ev     syscall.Handle
		status api.Status
		t0     can.Time
		t0val  int64
	}
}

func (*driver) Scan() (list []can.Name) {
	for _, bus := range hwList {
		if !bus.pnp {
			continue
		}
		for i, ch := range bus.channels {
			if ch.Available() {
				ch.Initialize(api.Baud500K, 0, 0, 0)
				disp := ch.DisplayName()
				ch.Uninitialize()
				list = append(list, can.Name{
					ID:      bus.name + strconv.Itoa(i+1),
					Display: disp,
					Driver:  "pcan",
				})
			}
		}
	}
	return
}

func (*driver) Open(devName string, options ...interface{}) (cd can.Device, err error) {
	defer wrapErr("open", &err)
	d := new(dev)

	b, i, err := hwList.lookupName(devName)
	if err != nil {
		return
	}
	h := b.channels[i]

	if d.receive.ev, err = win.CreateEvent(!win.EvManualReset, !win.EvInitiallyOn); err != nil {
		return
	}

	if h.InUse() {
		err = errors.New("channel in use")
		return
	}
	if !h.Available() {
		err = errors.New("channel not available")
		return
	}

	bitrate, err := scanOptions(options)
	if err != nil {
		return
	}
	if err = h.Initialize(api.Baudrate(bitrate), 0, 0, 0).Err(); err != nil {
		return
	}

	if err = h.SetValue(api.BusoffAutoreset, true).Err(); err != nil {
		return
	}

	if err = h.SetValue(api.ReceiveEvent, d.receive.ev).Err(); err != nil {
		return
	}

	d.h = h
	d.name = can.Name{
		ID:      b.name + strconv.Itoa(i+1),
		Driver:  "pcan",
		Display: h.DisplayName(),
	}

	cd = d
	return
}

func (d *dev) Name() can.Name {
	return d.name
}

func (d *dev) Read(buf []can.Msg) (n int, err error) {
	var m api.Msg
	var ts api.TimeStamp

	defer wrapErr("read", &err)

	prevSt := d.receive.status
	defer func() {
		d.receive.status = prevSt
	}()

	block := false
	hasBlocked := false
	for n < len(buf) {
		st := d.h.ReadMsg(&m, &ts)

	reEval:
		switch {
		case st == 0:
			µs := int64(ts.Micros) + 1000*int64(ts.Millis) + 0xFFFFFFFF*1000*int64(ts.Overflow)
			st = d.decode(&buf[n], &m, µs)
			if st != 0 {
				// It seems that this path is never entered, i.e. apparently
				// ReadMsg won't return api.OK if it stored a status message into
				// its api.Msg argument with as status different from api.OK
				goto reEval
			}
			prevSt = st
			n++

		case st.Test(api.ErrINITIALIZE):
			err = io.EOF
			return

		case st.Test(api.ErrBUSHEAVY | api.ErrBUSOFF):
			st &= api.ErrBUSHEAVY | api.ErrBUSOFF
			if st != prevSt {
				buf[n] = can.Msg{
					Flags: errFlagsMap.Decode(int(st)) | can.StatusMsg,
				}
				n++
				prevSt = st
				return
			}
			hasBlocked = false
			block = true

		case st.Test(api.ErrBUSLIGHT):
			// ignore

		case st.Test(api.ErrQRCVEMPTY):
			// It may be possible that at this point the receive queue
			// is empty, `n' is still zero, and the receive event is
			// in a signaled state (because it has been triggered again
			// during the last read call, but not yet handled).
			//
			// In this case WaitForSingleObject will return immediately,
			// and -- because of the queue still being empty -- this
			// branch will be entered a second time. Hence the test
			// for n > 0 below, to avoid making Read return without
			// having read anything.

			if hasBlocked && n > 0 {
				return
			}
			block = true

		default:
			err = st
			return
		}
		if block {
			ev, err1 := syscall.WaitForSingleObject(d.receive.ev, syscall.INFINITE)
			switch ev {
			case syscall.WAIT_OBJECT_0:
				hasBlocked = true
			case syscall.WAIT_FAILED:
				err = errors.New("pcan: read: WaitForSingleObject failed: " + err1.Error())
			default:
				err = errors.New("pcan: read: WaitForSingleObject: unexpected error")
			}
			block = false
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

retry:
	st := d.h.WriteMsg(&m)
	switch {
	case st == 0:
	case st.Test(api.ErrBUSOFF) || st.Test(api.ErrQXMTFULL):
		// simulate blocking
		time.Sleep(100 * time.Millisecond)
		goto retry
	default:
		err = st
	}

	return
}

func (d *dev) Close() (err error) {
	err = d.h.Uninitialize().Err()
	win.SetEvent(d.receive.ev)
	syscall.CloseHandle(d.receive.ev)
	wrapErr("close", &err)
	return
}

func (d *dev) Version() (v can.Version) {
	apiVer, st := d.h.StringVal(api.ApiVersion)
	if st == api.OK {
		f := strings.FieldsFunc(apiVer, func(r rune) bool {
			switch r {
			case '.', ',', ' ':
				return true
			}
			return false
		})
		v.Api = strings.Join(f, ".")
	}
	chanVer, st := d.h.StringVal(api.ChanVersion)
	if st == api.OK {
		v.Device = chanVer
	}
	return
}
