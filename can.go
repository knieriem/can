// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can

import (
	"errors"
	"strings"
)

type Error string

func (e Error) Error() string {
	return "can: " + string(e)
}

type Driver interface {
	Name() string
	//	Version() string
	Open(env *Env, name string, conf *Config) (Device, error)
	Scan() []DeviceInfo
}

var drvlist []Driver

func RegisterDriver(drv Driver) {
	if drv == UnsupportedDriver {
		return
	}
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
	Info() *DeviceInfo

	Close() error
}

type DeviceInfo struct {
	ID string

	Model  string
	Device string
	Driver string

	SystemDriver        string
	SystemDriverVersion string

	APIVersion string
	Firmware   string
	SerialNum  string
}

func (di *DeviceInfo) String() string {
	return di.Driver + ":" + di.ID
}

func (di *DeviceInfo) Format(idSep, itemSep, end string) string {
	var item []string
	if di.Model != "" {
		item = append(item, di.Model)
	}
	if di.Device != "" && di.Device != di.ID {
		item = append(item, di.Device)
	}
	if di.SystemDriver != "" {
		item = append(item, di.SystemDriver)
	}
	s := ""
	if idSep != "<OMIT ID>" {
		s += di.String() + idSep
	}
	return s + strings.Join(item, itemSep) + end
}

type Option func(*openProps)

type openProps struct {
	conf *Config
}

func WithConfig(conf *Config) Option {
	return func(p *openProps) {
		p.conf = conf.clone()
	}
}

// Open tries to open a CAN device matching the device specification.
// The deviceSpec has the syntax
//
//	[ driverName [ ":" deviceName ] { "," ctlString } ]
//
// The syntax suggests that "" is a valid input: It will try
// to open any available CAN adapter with driver dependent default settings.
// The comma separated ctl strings will be processed by [ParseConfig].
// On success, a Device instance will be returned, else an error.
func Open(deviceSpec string, opts ...Option) (dev Device, err error) {
	var p openProps
	var env Env

	for _, o := range opts {
		o(&p)
	}

	env.BufPool = newSimpleBufPool(16)

	if p.conf == nil {
		f := strings.Split(deviceSpec, ",")
		if len(f) > 1 {
			optFields := f[1:]
			c, err := ParseConfig(optFields...)
			if err != nil {
				return nil, err
			}
			p.conf = c
			deviceSpec = f[0]
		}
	}

	f := strings.SplitN(deviceSpec, ":", 2)
	name := ""

	drvName := f[0]
	if drvName == "" {
		for _, drv := range drvlist {
			dev, err = drv.Open(&env, name, p.conf)
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
			return drv.Open(&env, name, p.conf)
		}
	}
	err = Error("driver not found: " + drvName)
	return
}

func Scan() (list []DeviceInfo) {
	for _, drv := range drvlist {
		list = append(list, drv.Scan()...)
	}
	return
}

type Env struct {
	BufPool DataBufPool
}

type Unversioned struct{}

func (Unversioned) Info() *DeviceInfo { return &DeviceInfo{} }

var UnsupportedDriver Driver = unsupported{}

type unsupported struct{}

func (unsupported) Name() string { return "unsupported" }
func (unsupported) Open(_ *Env, name string, conf *Config) (Device, error) {
	return nil, errors.New("not supported")
}
func (unsupported) Scan() []DeviceInfo { return nil }

// ErrTxQueueFull is returned when a Msg could not be added
// to the devices' transmit queue. On Linux, it is returned
// in case of ENOBUFS. Normally this error is caused by a wiring
// problem, or if no CAN node is present on the bus.
var ErrTxQueueFull = Error("tx queue full")
