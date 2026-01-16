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

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/pcan/internal/filter"
	"golang.org/x/sys/windows"
)

//sys initialize(h Handle, btr0btr1 Baudrate, hw HwType, ioport uint32, intr uint16) (status Status) = pcanbasic.CAN_Initialize
//sys initializeFD(h Handle, bitrateFD *byte) (status Status) = pcanbasic.CAN_InitializeFD
//sys uninitialize(h Handle) (status Status) = pcanbasic.CAN_Uninitialize

//sys reset(h Handle) (status Status) = pcanbasic.CAN_Reset
//sys status(h Handle) (status Status) = pcanbasic.CAN_GetStatus

//sys readMsg(h Handle, buf *Msg, ts *TimeStamp) (status Status) = pcanbasic.CAN_Read
//sys readMsgFD(h Handle, buf *MsgFD, ts *TimeStampFD) (status Status) = pcanbasic.CAN_ReadFD
//sys writeMsg(h Handle, buf *Msg) (status Status) = pcanbasic.CAN_Write
//sys writeMsgFD(h Handle, buf *MsgFD) (status Status) = pcanbasic.CAN_WriteFD

//sys filterMsgs(h Handle, fromID uint32, toID uint32, mode Mode) (status Status) = pcanbasic.CAN_FilterMessages

//sys getValue(h Handle, p byte, buf uintptr, size uintptr) (s Status) = pcanbasic.CAN_GetValue
//sys setValue(h Handle, p byte, buf uintptr, size uintptr) (s Status) = pcanbasic.CAN_SetValue

func (h Handle) Initialize(btr0btr1 Baudrate, hw HwType, ioPort uint32, intr uint16) Status {
	return initialize(h, btr0btr1, hw, ioPort, intr)
}

func (h Handle) InitializeFD(bitrateFD string) error {
	p0, err := syscall.BytePtrFromString(bitrateFD)
	if err != nil {
		return err
	}

	return initializeFD(h, p0).Err()
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

func (m *Msg) Data() []byte {
	n := int(m.LEN)
	if n > len(m.DATA) {
		n = len(m.DATA)
	}
	return m.DATA[:n]
}

func (h Handle) ReadMsgFD(m *MsgFD, ts *TimeStampFD) Status {
	return readMsgFD(h, m, ts)
}

func (m *MsgFD) Data() []byte {
	n := int(m.DLC)
	if n > 8 {
		i := n - 9
		if i >= len(can.ValidFDSizes) {
			i = len(can.ValidFDSizes)
		}
		n = can.ValidFDSizes[i]
	}
	if n > len(m.DATA) {
		return m.DATA[:]
	}
	return m.DATA[:n]
}

func (h Handle) WriteMsg(m *Msg) Status {
	return writeMsg(h, m)
}

func (h Handle) WriteMsgFD(m *MsgFD) Status {
	return writeMsgFD(h, m)
}

func (h Handle) FilterMsgs(filters []can.MsgFilter) error {

	if len(filters) == 0 {
		return nil
	}

	st := h.SetValue(MsgFilter, FilterClose)
	if st != OK {
		return st
	}

	f := make(filter.Filter, 0, len(filters))

	f.Add(filters, false)
	err := h.applyFilters(f, ModeStandard)
	if err != nil {
		return err
	}

	f = f[:0]
	f.Add(filters, true)
	return h.applyFilters(f, ModeExtended)
}

func (h Handle) applyFilters(filters filter.Filter, mode Mode) error {

	for i := range filters {
		iv := &filters[i]
		st := filterMsgs(h, iv.Start, iv.End, mode)
		if st != OK {
			return st
		}
	}
	return nil
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

func (p IntPar) set(h Handle, value interface{}) Status {
	var v int32
	switch x := value.(type) {
	case int:
		v = int32(x)
	case int32:
		v = x
	}
	return setValue(h, byte(p), uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
}

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
	v := i.(windows.Handle)
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

func AttachedDevices() []ChanInf {
	n, st := NoneBus.IntVal(AttachedChannelsCount)
	if st != OK || n <= 0 {
		return nil
	}
	buf := make([]ChanInf, n)

	p := uintptr(unsafe.Pointer(unsafe.SliceData(buf)))
	sz := uintptr(len(buf)) * unsafe.Sizeof(buf[0])

	if st = getValue(NoneBus, AttachedChannels, p, sz); st != OK {
		return nil
	}
	return buf
}

func (ci *ChanInf) Available() bool {
	return ci.Channel_condition == ChanAvailable
}

func (ci *ChanInf) Handle() Handle {
	return Handle(ci.Channel_handle)
}

func (ci *ChanInf) DeviceName() string {
	b := unsafe.Slice((*byte)(unsafe.Pointer(&ci.Device_name[0])), len(ci.Device_name))
	return windows.ByteSliceToString(b)
}

func (ci *ChanInf) DisplayName() string {
	name := ci.DeviceName()
	if name != "" {
		name += " "
	}
	name += "#" + strconv.Itoa(int(ci.Device_id))
	return name
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
		s = windows.ByteSliceToString(buf)
	} else {
		s = fmt.Sprintf("CAN_GetErrorText failed (0x%02x)", status)
	}
	return
}
