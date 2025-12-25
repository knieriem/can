// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"fmt"
	"log"
	"strconv"
	"syscall"
	"unsafe"
)

//sys initialize(h Handle, btr0btr1 Baudrate, hw HwType, ioport uint32, intr uint16) (status Status) = pcanbasic.CAN_Initialize
//sys uninitialize(h Handle) (status Status) = pcanbasic.CAN_Uninitialize

//sys reset(h Handle) (status Status) = pcanbasic.CAN_Reset
//sys status(h Handle) (status Status) = pcanbasic.CAN_GetStatus

//sys readMsg(h Handle, buf *Msg, ts *TimeStamp) (status Status) = pcanbasic.CAN_Read
//sys writeMsg(h Handle, buf *Msg) (status Status) = pcanbasic.CAN_Write

//sys filterMsgs(h Handle, fromID uint32, toID uint32, mode Mode) (status Status) = pcanbasic.CAN_FilterMessages

//sys getValue(h Handle, p byte, buf uintptr, size uintptr) (s Status) = pcanbasic.CAN_GetValue
//sys setValue(h Handle, p byte, buf uintptr, size uintptr) (s Status) = pcanbasic.CAN_SetValue

func (h Handle) Initialize(btr0btr1 Baudrate, hw HwType, ioPort uint32, intr uint16) Status {
	return initialize(h, btr0btr1, hw, ioPort, intr)
}

func (h Handle) Uninitialize() Status {
	return uninitialize(h)
}

func (h Handle) Reset() Status {
	return reset(h)
}

func (h Handle) Status() Status {
	return status(h)
}

func (h Handle) ReadMsg(m *Msg, ts *TimeStamp) Status {
	return readMsg(h, m, ts)
}

func (h Handle) WriteMsg(m *Msg) Status {
	return writeMsg(h, m)
}

type setter interface {
	set(Handle, interface{}) Status
}

func (h Handle) SetValue(p setter, v interface{}) Status {
	return p.set(h, v)
}

type BoolPar byte
type StringPar byte
type IntPar byte
type HandlePar byte

func (p StringPar) set(h Handle, i interface{}) Status {
	buf := []byte(i.(string))
	buf = append(buf, '\000')
	return setValue(h, byte(p), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
}

func (p BoolPar) set(h Handle, i interface{}) Status {
	var v int32 = ParamOff
	if b := i.(bool); b {
		v = ParamOn
	}
	return setValue(h, byte(p), uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
}

func (p HandlePar) set(h Handle, i interface{}) Status {
	v := i.(syscall.Handle)
	return setValue(h, byte(p), uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
}

func (h Handle) StringVal(p StringPar) (s string, st Status) {
	var buf = make([]byte, 256)
	if st = getValue(h, byte(p), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); st == OK {
		s = bufToString(buf)
	}
	return
}

func (h Handle) IntVal(p IntPar) (i int, st Status) {
	var cInt int32

	if st = getValue(h, byte(p), uintptr(unsafe.Pointer(&cInt)), unsafe.Sizeof(cInt)); st == OK {
		i = int(cInt)
	}
	return
}

func (h Handle) HandleVal(p HandlePar) (sh syscall.Handle, st Status) {
	st = getValue(h, byte(p), uintptr(unsafe.Pointer(&sh)), unsafe.Sizeof(sh))
	return
}

func (h Handle) BoolVal(p BoolPar) (b bool, st Status) {
	var v int32
	st = getValue(h, byte(p), uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	if st == 0 && v == ParamOn {
		b = true
	}
	return
}

func (h Handle) Available() (is bool) {
	v, st := h.IntVal(ChanCondition)
	if st == 0 && v == ChanAvailable {
		is = true
	}
	return
}

func (h Handle) InUse() (is bool) {
	v, st := h.IntVal(ChanCondition)
	if st == 0 && v == ChanOccupied {
		is = true
	}
	return
}

func (h Handle) DisplayName() (name string) {
	s, st := h.StringVal(HardwareName)
	if st == 0 {
		name = s
	} else {
		log.Println("DI", st)
	}
	i, st := h.IntVal(DeviceId)
	if st == 0 {
		if name != "" {
			name += " "
		}
		name += "#" + strconv.Itoa(i)
	}
	return
}

//sys errorText(err Status, lang uint16, buf *byte) (s Status) = pcanbasic.CAN_GetErrorText

const (
	neutral = 0
	english = 9
)

func (err Status) Error() (s string) {
	var buf = make([]byte, 256)

	status := errorText(err, english, &buf[0])
	if status == OK {
		s = bufToString(buf)
	} else {
		s = fmt.Sprintf("CAN_GetErrorText failed (0x%02x)", status)
	}
	return
}
