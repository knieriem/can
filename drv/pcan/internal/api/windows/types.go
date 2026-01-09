// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package api

/*
#include "windows.h"
#define CAN_Initialize(a, b, c, d, e) CAN_Dummy(void)
#include "PCANBasic.h"

*/
import "C"

type MsgType C.TPCANMessageType
type HwType C.TPCANDevice
type Baudrate C.TPCANBaudrate
type Handle C.TPCANHandle
type Mode C.TPCANMode

type Msg C.TPCANMsg
type MsgFD C.TPCANMsgFD

type TimeStamp C.TPCANTimestamp
type TimeStampFD C.TPCANTimestampFD

type ChanInf C.TPCANChannelInformation
