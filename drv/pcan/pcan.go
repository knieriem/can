// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pcan

import (
	"encoding/binary"
	"errors"
	"strconv"

	"can"
	"can/drv"
	api "can/drv/pcan/api"
)

const (
	defaultBitrate = api.Baud500K
)

var builtinBitrates = map[can.Bitrate]uint16{
	1000000: 0x0014,
	800000:  0x0016,
	500000:  0x001C,
	250000:  0x011C,
	125000:  0x031C,
	100000:  0x432F,
	95000:   0xC34E,
	83000:   0x852B,
	50000:   0x472F,
	47000:   0x1414,
	33000:   0x8B2F,
	20000:   0x532F,
	10000:   0x672F,
	5000:    0x7F7F,
}

func init() {
	can.RegisterDriver(new(driver))
}

type driver struct {
}

func (*driver) Name() string {
	return "pcan"
}

func scanOptions(list []interface{}) (bitrate uint16, err error) {
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
		}
	}
	return
}

type busList []*bus

func (buses busList) lookup(name string) *bus {
	for _, b := range buses {
		if b.name == name {
			return b
		}
	}
	return nil
}

var errBusNotFound = errors.New("no such bus")

func (buses busList) lookupName(name string) (b *bus, i int, err error) {
	switch len(name) {
	case 0:
		for _, bus := range buses {
			if !bus.canAutoDetect() {
				continue
			}
			i = bus.getFirstAvail()
			if i != -1 {
				b = bus
				return
			}
		}
		err = errors.New("no channels available")

	case 1, 2:
		err = errBusNotFound

	case 3:
		b = buses.lookup(name)
		if b == nil {
			err = errBusNotFound
			break
		}
		if !b.canAutoDetect() {
			err = errors.New("channel detection not available for selected bus")
			break
		}
		i = b.getFirstAvail()
		if i == -1 {
			err = errors.New("no channels available on bus")
		}
	default:
		b = buses.lookup(name[:3])
		if b == nil {
			err = errBusNotFound
			break
		}
		i, err = strconv.Atoi(name[3:])
		if err != nil {
			break
		}
		i--
		if i < 0 || i >= b.nChan() {
			err = errors.New("channel not found on bus")
		}
	}
	return
}

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

var errFlagsMap = drv.FlagsMap{
	{can.ErrorActive, int(api.ErrBUSLIGHT)},
	{can.ErrorPassive, int(api.ErrBUSHEAVY)},
	{can.BusOff, int(api.ErrBUSOFF)},
	{can.DataOverrun, int(api.ErrOVERRUN)},
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
		dst.Len = 0
		return
	}
	dst.Id = m.ID
	dst.Flags = msgFlagsMap.Decode(int(m.MSGTYPE))
	dst.Len = int(m.LEN)
	copy(dst.Data[:], m.DATA[:dst.Len])
	return
}

func encode(dst *api.Msg, src *can.Msg) {
	dst.ID = src.Id
	dst.LEN = byte(src.Len)
	dst.MSGTYPE = byte(msgFlagsMap.Encode(src.Flags))
	copy(dst.DATA[:src.Len], src.Data[:src.Len])
}

type wrappedError struct {
	error
	fnName string
}

func (e wrappedError) Error() string {
	return "pcan: " + e.fnName + ": " + e.error.Error()
}

func wrapErr(fnName string, pErr *error) {
	err := *pErr
	if err != nil {
		*pErr = &wrappedError{err, fnName}
	}
}

func (b *bus) nChan() int {
	return len(b.channels)
}
