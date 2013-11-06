// Created by api-defs.sh - DO NOT EDIT
// linux/api-defs.sh

package api

import (
	"unsafe"
)

func (f Fd) Init(p *Init) error {
	return ioctl(uintptr(f), ioctlINIT, uintptr(unsafe.Pointer(p)))
}

func (f Fd) WriteMsg(p *Msg) error {
	return ioctl(uintptr(f), ioctlWRITE_MSG, uintptr(unsafe.Pointer(p)))
}

func (f Fd) ReadMsg(p *RMsg) error {
	return ioctl(uintptr(f), ioctlREAD_MSG, uintptr(unsafe.Pointer(p)))
}

func (f Fd) status(p *statusPar) error {
	return ioctl(uintptr(f), ioctlGET_STATUS, uintptr(unsafe.Pointer(p)))
}

func (f Fd) Diag(p *Diag) error {
	return ioctl(uintptr(f), ioctlDIAG, uintptr(unsafe.Pointer(p)))
}

func (f Fd) SetBitrate(p *Btr0Btr1) error {
	return ioctl(uintptr(f), ioctlBTR0BTR1, uintptr(unsafe.Pointer(p)))
}

func (f Fd) ExtStatus(p *ExtStatus) error {
	return ioctl(uintptr(f), ioctlGET_EXT_STATUS, uintptr(unsafe.Pointer(p)))
}

func (f Fd) SetMsgFilter(p *MsgFilter) error {
	return ioctl(uintptr(f), ioctlMSG_FILTER, uintptr(unsafe.Pointer(p)))
}

func (f Fd) SetExtraParams(p *Params) error {
	return ioctl(uintptr(f), ioctlEXTRA_PARAMS, uintptr(unsafe.Pointer(p)))
}
