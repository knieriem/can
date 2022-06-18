// Code generated by 'go generate'; DO NOT EDIT.

package api

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var _ unsafe.Pointer

// Do the interface allocations only once for common
// Errno values.
const (
	errnoERROR_IO_PENDING = 997
)

var (
	errERROR_IO_PENDING error = syscall.Errno(errnoERROR_IO_PENDING)
	errERROR_EINVAL     error = syscall.EINVAL
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return errERROR_EINVAL
	case errnoERROR_IO_PENDING:
		return errERROR_IO_PENDING
	}
	// TODO: add more here, after collecting data on the common
	// error values see on Windows. (perhaps when running
	// all.bat?)
	return e
}

var (
	modpcanbasic = windows.NewLazySystemDLL("pcanbasic.dll")

	procCAN_FilterMessages = modpcanbasic.NewProc("CAN_FilterMessages")
	procCAN_GetErrorText   = modpcanbasic.NewProc("CAN_GetErrorText")
	procCAN_GetStatus      = modpcanbasic.NewProc("CAN_GetStatus")
	procCAN_GetValue       = modpcanbasic.NewProc("CAN_GetValue")
	procCAN_Initialize     = modpcanbasic.NewProc("CAN_Initialize")
	procCAN_Read           = modpcanbasic.NewProc("CAN_Read")
	procCAN_Reset          = modpcanbasic.NewProc("CAN_Reset")
	procCAN_SetValue       = modpcanbasic.NewProc("CAN_SetValue")
	procCAN_Uninitialize   = modpcanbasic.NewProc("CAN_Uninitialize")
	procCAN_Write          = modpcanbasic.NewProc("CAN_Write")
)

func filterMsgs(h Handle, fromID uint32, toID uint32, mode Mode) (status Status) {
	r0, _, _ := syscall.Syscall6(procCAN_FilterMessages.Addr(), 4, uintptr(h), uintptr(fromID), uintptr(toID), uintptr(mode), 0, 0)
	status = Status(r0)
	return
}

func errorText(err Status, lang uint16, buf *byte) (s Status) {
	r0, _, _ := syscall.Syscall(procCAN_GetErrorText.Addr(), 3, uintptr(err), uintptr(lang), uintptr(unsafe.Pointer(buf)))
	s = Status(r0)
	return
}

func status(h Handle) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_GetStatus.Addr(), 1, uintptr(h), 0, 0)
	status = Status(r0)
	return
}

func getValue(h Handle, p byte, buf uintptr, size uintptr) (s Status) {
	r0, _, _ := syscall.Syscall6(procCAN_GetValue.Addr(), 4, uintptr(h), uintptr(p), uintptr(buf), uintptr(size), 0, 0)
	s = Status(r0)
	return
}

func initialize(h Handle, btr0btr1 Baudrate, hw HwType, ioport uint32, intr uint16) (status Status) {
	r0, _, _ := syscall.Syscall6(procCAN_Initialize.Addr(), 5, uintptr(h), uintptr(btr0btr1), uintptr(hw), uintptr(ioport), uintptr(intr), 0)
	status = Status(r0)
	return
}

func readMsg(h Handle, buf *Msg, ts *TimeStamp) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_Read.Addr(), 3, uintptr(h), uintptr(unsafe.Pointer(buf)), uintptr(unsafe.Pointer(ts)))
	status = Status(r0)
	return
}

func reset(h Handle) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_Reset.Addr(), 1, uintptr(h), 0, 0)
	status = Status(r0)
	return
}

func setValue(h Handle, p byte, buf uintptr, size uintptr) (s Status) {
	r0, _, _ := syscall.Syscall6(procCAN_SetValue.Addr(), 4, uintptr(h), uintptr(p), uintptr(buf), uintptr(size), 0, 0)
	s = Status(r0)
	return
}

func uninitialize(h Handle) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_Uninitialize.Addr(), 1, uintptr(h), 0, 0)
	status = Status(r0)
	return
}

func writeMsg(h Handle, buf *Msg) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_Write.Addr(), 2, uintptr(h), uintptr(unsafe.Pointer(buf)), 0)
	status = Status(r0)
	return
}
