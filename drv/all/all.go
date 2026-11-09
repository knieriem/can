// Package all registers all drivers of this module that are available
// on the platform, so that a program picks up new drivers by updating
// the module:
//
//	all.Register()
//
// To use a driver with other options, register it after calling
// Register; it replaces the default driver of the same name.
//
// For compatibility, importing the package also registers the
// drivers; a future version will not.
package all

import (
	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/canrpc"
	"github.com/knieriem/can/drv/pcan"
	"github.com/knieriem/can/drv/socketcan"
)

// Register registers all drivers of this module that are available
// on the platform.
func Register() {
	can.RegisterDriver(canrpc.Driver)
	can.RegisterDriver(pcan.Driver)
	can.RegisterDriver(socketcan.Driver)
}

func init() {
	Register()
}
