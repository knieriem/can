// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can

import (
	"strings"
)

type Error string

func (e Error) Error() string {
	return "can: " + string(e)
}

type Driver interface {
	Name() string
	//	Version() string
	Open(name string, options ...interface{}) (Device, error)
	Scan() []Name
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

	ID() string
	Name() Name
	Version() Version

	Close() error
}

type Name struct {
	ID      string
	Display string
	Device  string
	Driver  string
}

func (n *Name) String() string {
	return n.Driver + ":" + n.ID
}

func (n *Name) Format(idSep, itemSep, end string) string {
	var item []string
	if n.Display != "" {
		item = append(item, n.Display)
	}
	if n.Device != "" {
		item = append(item, n.Device)
	}
	s := ""
	if idSep != "<OMIT ID>" {
		s += n.String() + idSep
	}
	return s + strings.Join(item, itemSep) + end
}

type Version struct {
	Device    string
	Driver    string
	Api       string
	SerialNum string
}

//
//	driverName:deviceName:[,option][,option2]
//
//	name	Go driver name
func Open(deviceName string, options ...string) (dev Device, err error) {
	f := strings.Split(deviceName, ",")
	optFields := f[1:]
	optList, err := parseOptions(optFields)
	if err != nil {
		return
	}
	deviceName = f[0]

	f = strings.SplitN(deviceName, ":", 2)
	name := ""

	drvName := f[0]
	if drvName == "" {
		for _, drv := range drvlist {
			dev, err = drv.Open(name, optList...)
			if err == nil {
				return
			}
		}
		err = Error("no device found")
		return
	}

	if len(f) == 2 {
		name = f[1]
	}

	for _, drv := range drvlist {
		if drv.Name() == drvName {
			return drv.Open(name, optList...)
		}
	}
	err = Error("driver not found: " + drvName)
	return
}

func Scan() (list []Name) {
	for _, drv := range drvlist {
		list = append(list, drv.Scan()...)
	}
	return
}

type Unversioned struct{}

func (Unversioned) Name() Name       { return Name{} }
func (Unversioned) Version() Version { return Version{} }
