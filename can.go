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
	Open(name string, conf *Config) (Device, error)
	Scan() []DeviceInfo
}

var drvlist []Driver

// RegisterDriver makes drv available to Open and Scan. A driver with
// the name of an earlier one replaces it, so that a program can import
// drv/all and still register a driver with its own options.
func RegisterDriver(drv Driver) {
	if drv == UnsupportedDriver {
		return
	}
	for i, d := range drvlist {
		if d.Name() == drv.Name() {
			drvlist[i] = drv
			return
		}
	}
	drvlist = append(drvlist, drv)
}

// A Device is an open CAN channel.
//
// Read may be called by one goroutine at a time. WriteMsg may be
// called concurrently with Read and with itself. Close may be called
// at any time, from any goroutine, and more than once: it unblocks
// pending Read and WriteMsg calls, which then return an error
// wrapping [ErrClosed]; so do all calls after Close, including
// further Close calls. ID and Info do not change after Open.
type Device interface {
	// Read blocks until at least one message is available,
	// stores up to len(buf) messages into buf, and returns their
	// number. It reuses the payload storage of buf's messages.
	// An error means the device can no longer be read.
	Read(buf []Msg) (n int, err error)

	// WriteMsg queues m for transmission. It does not retain m.
	// If the transmit queue is full, it returns an error wrapping
	// [ErrTxQueueFull]; the caller decides whether to retry.
	WriteMsg(m *Msg) error

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
	conf   *Config
	status statusLevel
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
//
// Read delivers no status messages, unless WithStatusMsgs or
// WithBusErrors is given.
func Open(deviceSpec string, opts ...Option) (dev Device, err error) {
	var p openProps

	for _, o := range opts {
		o(&p)
	}

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
			// Drivers may modify the config; give each its own copy.
			dev, err = drv.Open(name, p.conf.clone())
			if err == nil {
				return withStatusFilter(dev, p.status), nil
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
			dev, err = drv.Open(name, p.conf)
			if err != nil {
				return nil, err
			}
			return withStatusFilter(dev, p.status), nil
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

type Unversioned struct{}

func (Unversioned) Info() *DeviceInfo { return &DeviceInfo{} }

var UnsupportedDriver Driver = unsupported{}

type unsupported struct{}

func (unsupported) Name() string { return "unsupported" }
func (unsupported) Open(name string, conf *Config) (Device, error) {
	return nil, errors.New("not supported")
}
func (unsupported) Scan() []DeviceInfo { return nil }

// ErrClosed is returned, wrapped, by a Device's methods
// once the device has been closed.
var ErrClosed = Error("device closed")

// ErrBusOff is returned, wrapped, by WriteMsg when the controller is
// in the bus-off state and cannot transmit.
var ErrBusOff = Error("bus-off")

// ErrTxQueueFull is returned when a Msg could not be added
// to the devices' transmit queue. On Linux, it is returned
// in case of ENOBUFS. Normally this error is caused by a wiring
// problem, or if no CAN node is present on the bus.
var ErrTxQueueFull = Error("tx queue full")
