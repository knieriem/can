//go:build !linux

package socketcan

import (
	"github.com/knieriem/can"
)

var Driver = can.UnsupportedDriver
