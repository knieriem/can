// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pcan

import (
	"encoding/binary"
	"errors"
	"strconv"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv"
	api "github.com/knieriem/can/drv/pcan/internal/api"
	"github.com/knieriem/can/timing"
)

const (
	defaultBitrate = api.Baud500K
)

var builtinBitrates = map[uint32]uint16{
	1000000: api.Baud1M,
	800000:  api.Baud800K,
	500000:  api.Baud500K,
	250000:  api.Baud250K,
	125000:  api.Baud125K,
	100000:  api.Baud100K,
	95000:   api.Baud95K,
	83000:   api.Baud83K,
	50000:   api.Baud50K,
	47000:   api.Baud47K,
	33000:   api.Baud33K,
	20000:   api.Baud20K,
	10000:   api.Baud10K,
	5000:    api.Baud5K,
}

func init() {
	if driverPresent() {
		can.RegisterDriver(new(driver))
	}
}

type driver struct {
}

func (*driver) Name() string {
	return "pcan"
}

var DevSpecFD = timing.Controller{
	Nominal: timing.Constraints{
		TSeg1Max:     256,
		TSeg2Max:     128,
		SJWMax:       128,
		PrescalerMin: 1,
		PrescalerMax: 1024,
	},
	Data: &timing.Constraints{
		TSeg1Max:     32,
		TSeg2Max:     16,
		SJWMax:       16,
		PrescalerMin: 1,
		PrescalerMax: 1024,
	},
}

func timingConf(c *can.Config) (tc uint16, err error) {
	if c == nil {
		return defaultBitrate, nil
	}
	if v := c.Nominal.Bitrate; v != 0 {
		if tc, ok := builtinBitrates[v]; ok {
			return tc, nil
		} else {
			return 0, errors.New("bitrate not supported")
		}
	}
	return defaultBitrate, nil
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
		dst.SetData(nil)
		return
	}
	dst.Id = m.ID
	dst.Flags = msgFlagsMap.Decode(int(m.MSGTYPE))
	data := dst.Data()[:m.LEN]
	copy(data, m.DATA[:])
	dst.SetData(data)
	return
}

func encode(dst *api.Msg, src *can.Msg) {
	dst.ID = src.Id
	data := src.Data()
	dst.LEN = uint8(len(data))
	dst.MSGTYPE = byte(msgFlagsMap.Encode(src.Flags))
	copy(dst.DATA[:], data)
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

func (d *dev) ID() string {
	return "pcan:" + d.name.ID
}
