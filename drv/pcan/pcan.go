// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux || windows

package pcan

import (
	"errors"
	"strconv"

	"github.com/knieriem/can"
	api "github.com/knieriem/can/drv/pcan/internal/api"
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

// Driver is the pcan driver. Register it with can.RegisterDriver,
// or use package drv/all.
var Driver can.Driver = new(driver)

// For compatibility, importing the package registers Driver,
// if the system driver is present. A future version will not.
func init() {
	if driverPresent() {
		can.RegisterDriver(Driver)
	}
}

type driver struct {
}

func (*driver) Name() string {
	return "pcan"
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

// status translates a PCAN status word.
func status(st api.Status) can.Status {
	var s can.Status
	switch {
	case st&api.ErrBUSOFF != 0:
		s.State = can.BusOff
	case st&errBusPassive != 0:
		s.State = can.ErrorPassive
	case st&(api.ErrBUSHEAVY|api.ErrBUSLIGHT) != 0:
		s.State = can.ErrorWarning // BUSHEAVY is also named BUSWARNING
	}
	if st&(api.ErrOVERRUN|api.ErrQOVERRUN) != 0 {
		s.Events |= can.RxOverflow
	}
	return s
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

func (e wrappedError) Unwrap() error {
	return e.error
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
	return "pcan:" + d.info.ID
}
