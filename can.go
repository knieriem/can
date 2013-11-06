// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can

import (
	"errors"
	"strings"
)

type Driver interface {
	Name() string
	//	Version() string
	Open(name string) (Device, error)
	Scan() []string
}

var drvlist []Driver

func RegisterDriver(drv Driver) {
	drvlist = append(drvlist, drv)
}

// The Device interface gives access to a CAN Device.
// Read and Write calls will block if no messages are
// available to be read or if the transmit buffer
// of the driver is full.
type Device interface {
	Read([]Msg) (n int, err error)

	// Writes a message into the driver transmit buffer.
	// The ownership of the message will not be taken.
	WriteMsg(*Msg) error

	// As an alternative to WriteMsg, Write can be used
	// if more than one message should be handed over
	// to the driver at once (if the driver is able to do that).
	Write([]Msg) (n int, err error)

	DriverVersion() string

	Close() error
}

//
//	driverName:deviceName:[,option][,option2]
//
//	name	Go driver name
func Open(deviceName string, ctl string) (dev Device, err error) {
	f := strings.SplitN(deviceName, ":", 2)
	name := ""

	if f[0] == "" {
		for _, drv := range drvlist {
			dev, err = drv.Open(name)
			if err == nil {
				return
			}
		}
		err = errors.New("no device found")
		return
	}

	if len(f) == 2 {
		name = f[1]
	}

	for _, drv := range drvlist {
		if drv.Name() == f[0] {
			return drv.Open(name)
		}
	}
	err = errors.New("driver not found: " + f[0])
	return
}

func Scan() (list []string) {
	for _, drv := range drvlist {
		name := drv.Name()
		for _, s := range drv.Scan() {
			if s == "" {
				list = append(list, name)
			} else {
				list = append(list, name+":"+s)
			}
		}
	}
	return
}

type Unversioned struct{}

func (Unversioned) DriverVersion() string { return "" }
