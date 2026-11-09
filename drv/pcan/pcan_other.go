//go:build !linux && !windows

package pcan

import "github.com/knieriem/can"

// Driver is not supported on this platform.
var Driver = can.UnsupportedDriver
