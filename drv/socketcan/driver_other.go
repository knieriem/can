//go:build !linux

package socketcan

import (
	"github.com/knieriem/can"
)

var Driver = can.UnsupportedDriver

func NewDriver(...DriverOption) can.Driver {
	return Driver
}

type DriverOption func()

func WithPrivilegedUtil() DriverOption {
	return nil
}
