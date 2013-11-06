// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can4linux

import (
	"can"
	"can/drv"
)

const (
	remoteTransmissionReq = 1 << iota
	dataOverrun
	extFrameFormat
	_
	errorPassive
	busOff
	_
	receiveBufferOverflow

	ErrMask = errorPassive | busOff | receiveBufferOverflow | dataOverrun
)

var ErrFlagsMap = drv.FlagsMap{
	{can.ErrorPassive, errorPassive},
	{can.BusOff, busOff},
	{can.DataOverrun, dataOverrun},
	{can.ReceiveBufferOverflow, receiveBufferOverflow},
}

var MsgFlagsMap = drv.FlagsMap{
	{can.RTRMsg, remoteTransmissionReq},
	{can.ExtFrame, extFrameFormat},
}
