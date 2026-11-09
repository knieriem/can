// Package all registers all drivers of this module that are available
// on the platform, so that a program picks up new drivers by updating
// the module. Import it for its side effect:
//
//	import _ "github.com/knieriem/can/drv/all"
//
// To use a driver with other options, register it after importing all;
// it replaces the default driver of the same name.
package all

import (
	"github.com/knieriem/can"
	_ "github.com/knieriem/can/drv/canrpc"
	_ "github.com/knieriem/can/drv/pcan"
	"github.com/knieriem/can/drv/socketcan"
)

func init() {
	can.RegisterDriver(socketcan.Driver)
}
