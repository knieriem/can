// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pcan

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"golang.org/x/sys/windows"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv"
	"github.com/knieriem/can/drv/internal/canfd"
	"github.com/knieriem/can/drv/pcan/internal/api"
	"github.com/knieriem/can/timing"
)

// driverPresent reports whether PCANBasic.dll can be found. Without it,
// the generated API functions would panic, so Scan and Open check it.
var driverPresent = sync.OnceValue(func() bool {
	_, err := exec.LookPath("PCANBasic.dll")
	if err != nil {
		if errors.Is(err, exec.ErrDot) {
			err = nil
		}
	}
	return err == nil
})

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
	pnp = true
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
			api.USBBUS9,
			api.USBBUS10,
			api.USBBUS11,
			api.USBBUS12,
			api.USBBUS13,
			api.USBBUS14,
			api.USBBUS15,
			api.USBBUS16,
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
	},
}

func (buses busList) reverseLookup(h api.Handle) (*bus, int) {
	for _, b := range buses {
		i := slices.Index(b.channels, h)
		if i != -1 {
			return b, i
		}
	}
	return nil, -1
}

type dev struct {
	info    can.DeviceInfo
	h       api.Handle
	bus     *bus
	receive struct {
		ev     windows.Handle
		status api.Status
		t0     can.Time
		t0val  int64
	}

	msg    api.Msg
	fdMsg  api.MsgFD
	fdMode bool

	// closed is set by the first Close. readMu is held by Read
	// for its whole duration, so that Close can wait for a pending
	// Read to return before it closes the receive event handle.
	closed atomic.Bool
	readMu sync.Mutex
}

func (*driver) Scan() (list []can.DeviceInfo) {
	if !driverPresent() {
		return nil
	}
	d := api.AttachedDevices()
	for i := range d {
		ch := &d[i]
		if !ch.Available() {
			continue
		}
		bus, i := hwList.reverseLookup(ch.Handle())
		if bus == nil {
			continue
		}
		disp := ch.DisplayName()
		list = append(list, can.DeviceInfo{
			ID:     bus.name + strconv.Itoa(i+1),
			Model:  disp,
			Driver: "pcan",
		})
	}
	return
}

func (*driver) Open(devName string, conf *can.Config) (cd can.Device, err error) {
	if !driverPresent() {
		return nil, errors.New("pcan: PCANBasic.dll not found")
	}
	defer wrapErr("open", &err)
	d := new(dev)

	b, i, err := hwList.lookupName(devName)
	if err != nil {
		return
	}
	h := b.channels[i]

	if h.InUse() && !h.Available() {
		err = errors.New("channel in use")
		return
	}
	if !h.Available() {
		err = errors.New("channel not available")
		return
	}
	fdCapable := false
	feat, st := h.IntVal(api.ChanFeatures)
	if st == 0 {
		if feat&api.FeatureFdCapable != 0 {
			fdCapable = true
		}
	}
	btr0btr1, btStr, err := prepareBittiming(conf, fdCapable)
	if err != nil {
		return nil, err
	}
	if btStr != "" {
		if err = h.InitializeFD(btStr); err != nil {
			return nil, err
		}
	} else if err = h.Initialize(api.Baudrate(btr0btr1), 0, 0, 0).Err(); err != nil {
		return nil, err
	}

	if err = h.SetValue(api.BusoffAutoreset, true).Err(); err != nil {
		h.Uninitialize()
		return
	}

	if d.receive.ev, err = windows.CreateEvent(nil, 0, 0, nil); err != nil {
		h.Uninitialize()
		return
	}

	if err = h.SetValue(api.ReceiveEvent, d.receive.ev).Err(); err != nil {
		h.Uninitialize()
		windows.CloseHandle(d.receive.ev)
		return
	}

	if conf != nil && len(conf.MsgFilter) != 0 {
		err = h.FilterMsgs(conf.MsgFilter)
		if err != nil {
			h.Uninitialize()
			windows.CloseHandle(d.receive.ev)
			return nil, err
		}
	}

	d.h = h
	d.info = can.DeviceInfo{
		ID:     b.name + strconv.Itoa(i+1),
		Driver: "pcan",
		Model:  h.DisplayName(),
	}
	d.setupInfo()
	d.fdMode = btStr != ""

	cd = d
	return
}

func prepareBittiming(conf *can.Config, fdCapable bool) (tc uint16, btStr string, err error) {
	if conf == nil {
		return defaultBitrate, "", nil
	}

	fd, err := conf.ResolveFDMode(fdCapable)
	if err != nil {
		return 0, "", err
	}
	if fd {
		err := conf.ResolveBitTiming(&DevSpecFD)
		if err != nil {
			return 0, "", err
		}
		sb := new(strings.Builder)
		sb.WriteString("f_clock=80000000")
		formatBitTiming(sb, &conf.Nominal.BitTiming, "nom")
		if conf.Data.Valid {
			formatBitTiming(sb, &conf.Data.Value.BitTiming, "data")
		}
		return 0, sb.String(), nil
	}
	if v := conf.Nominal.Bitrate; v != 0 {
		if tc, ok := builtinBitrates[v]; ok {
			return tc, "", nil
		} else {
			return 0, "", errors.New("bitrate not supported")
		}
	}
	return defaultBitrate, "", nil
}

func formatBitTiming(w io.Writer, t *timing.BitTiming, prefix string) {
	fmt.Fprintf(w, ",%s_brp=%d,%s_tseg1=%d,%s_tseg2=%d,%s_sjw=%d",
		prefix, t.Prescaler,
		prefix, t.PropSeg+t.PhaseSeg1,
		prefix, t.PhaseSeg2,
		prefix, t.SJW)
}

func (d *dev) Info() *can.DeviceInfo {
	return &d.info
}

func (d *dev) Read(buf []can.Msg) (n int, err error) {
	defer wrapErr("read", &err)
	defer func() {
		if err != nil && d.closed.Load() {
			err = can.ErrClosed
		}
	}()

	d.readMu.Lock()
	defer d.readMu.Unlock()

	prevSt := d.receive.status
	defer func() {
		d.receive.status = prevSt
	}()

	block := false
	hasBlocked := false
	for n < len(buf) {
		if d.closed.Load() {
			if n > 0 {
				return n, nil
			}
			return 0, can.ErrClosed
		}
		st := api.OK
		err1 := d.readMsg(&buf[n])
		if err1 != nil {
			st2, ok := err1.(api.Status)
			if !ok {
				if n == 0 {
					return 0, err1
				}
				return n, nil
			}
			st = st2
		}
	reEval:
		switch {
		case st == 0:
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

		case st.Test(api.ErrANYBUSERR):
			st &= api.ErrANYBUSERR
			if st != prevSt {
				buf[n].SetStatus(status(st)) // keeps the payload storage
				n++
				prevSt = st
				return
			}
			hasBlocked = false
			block = true

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
			ev, err1 := windows.WaitForSingleObject(d.receive.ev, windows.INFINITE)
			switch ev {
			case syscall.WAIT_OBJECT_0:
				hasBlocked = true
			case syscall.WAIT_FAILED:
				return n, fmt.Errorf("WaitForSingleObject failed: %w", err1)
			default:
				return n, errors.New("WaitForSingleObject: unexpected result")
			}
			block = false
		}
	}
	return
}

func (d *dev) readMsg(m *can.Msg) error {
	if d.fdMode {
		var ts api.TimeStampFD
		am := &d.fdMsg
		st := d.h.ReadMsgFD(am, &ts)
		if st != api.OK {
			return st
		}
		return d.decode(m, am.ID, am.MSGTYPE, am.Data(), int64(ts))
	}

	var ts api.TimeStamp
	am := &d.msg
	st := d.h.ReadMsg(am, &ts)
	if st != api.OK {
		return st
	}
	µs := int64(ts.Micros) + 1000*int64(ts.Millis) + 0xFFFFFFFF*1000*int64(ts.Overflow)
	return d.decode(m, am.ID, am.MSGTYPE, am.Data(), µs)
}

// errBusPassive is the status bit for an error passive controller.
const errBusPassive = api.ErrBUSPASSIVE

var msgFlagsMap = drv.FlagsMap{
	{can.RTRMsg, api.MsgRtr},
	{can.ExtFrame, api.MsgExtended},
	{can.BRS, api.MsgBrs},
	{can.FDF, api.MsgFd},
	{can.ESI, api.MsgEsi},
}

func (d *dev) decode(dst *can.Msg, id uint32, msgType uint8, apiData []byte, µs int64) error {

	if d.receive.t0 == 0 {
		d.receive.t0 = can.Now()
		d.receive.t0val = µs
	}
	t := d.receive.t0 + can.Time(µs-d.receive.t0val)

	if msgType&api.MsgStatus != 0 {
		st := api.Status(binary.BigEndian.Uint32(apiData))
		dst.SetStatus(status(st))
		dst.Rx.Time = t
		return st.Err()
	}
	dst.Rx.Time = t
	dst.Id = id
	dst.Flags = msgFlagsMap.Decode(int(msgType))
	dst.SetData(apiData)
	return nil
}

func (d *dev) WriteMsg(cm *can.Msg) (err error) {
	defer wrapErr("write", &err)
	if d.closed.Load() {
		return can.ErrClosed
	}
	if cm.IsStatus() {
		return nil
	}

	var st api.Status
	if d.fdMode {
		var m api.MsgFD
		if err := encodeFD(&m, cm); err != nil {
			return err
		}
		st = d.h.WriteMsgFD(&m)
	} else {
		var m api.Msg
		encode(&m, cm)
		st = d.h.WriteMsg(&m)
	}
	switch {
	case st == api.OK:
		return nil
	case d.closed.Load():
		return can.ErrClosed
	case st.Test(api.ErrBUSOFF):
		return can.ErrBusOff
	case st.Test(api.ErrQXMTFULL):
		return can.ErrTxQueueFull
	}
	return st
}

func encodeFD(dst *api.MsgFD, src *can.Msg) error {
	dst.ID = src.Id
	data := src.Data()
	dlc, needsFD, err := canfd.DLC(len(data))
	if err != nil {
		return err
	}
	dst.DLC = dlc
	dst.MSGTYPE = byte(msgFlagsMap.Encode(src.Flags &^ can.ESI)) // ESI is not sent
	if needsFD || dst.MSGTYPE&api.MsgBrs != 0 {
		dst.MSGTYPE |= api.MsgFd
	}
	copy(dst.DATA[:], data)
	return nil
}

func (d *dev) Close() (err error) {
	defer wrapErr("close", &err)
	if d.closed.Swap(true) {
		return can.ErrClosed
	}
	err = d.h.Uninitialize().Err()

	// Wake a Read waiting for the receive event, and let it return
	// before closing the handle: closing a handle while a wait on it
	// is pending is undefined behavior.
	windows.SetEvent(d.receive.ev)
	d.readMu.Lock()
	windows.CloseHandle(d.receive.ev)
	d.readMu.Unlock()
	return err
}

func (d *dev) setupInfo() {
	apiVer, st := d.h.StringVal(api.ApiVersion)
	if st == api.OK {
		f := strings.FieldsFunc(apiVer, func(r rune) bool {
			switch r {
			case '.', ',', ' ':
				return true
			}
			return false
		})
		d.info.APIVersion = strings.Join(f, ".")
	}
	chanVer, st := d.h.StringVal(api.ChanVersion)
	if st == api.OK {
		d.info.SystemDriverVersion = chanVer
	}
	fwVer, st := d.h.StringVal(api.FirmwareVersion)
	if st == api.OK {
		d.info.Device = fwVer
	}
}
