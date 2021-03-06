// mksyscall_windows.pl -l32 api_windows.go
// MACHINE GENERATED BY THE COMMAND ABOVE; DO NOT EDIT

package api

import "unsafe"
import "syscall"

var (
	modpcanbasic = syscall.NewLazyDLL("pcanbasic.dll")

	procCAN_Initialize     = modpcanbasic.NewProc("CAN_Initialize")
	procCAN_Uninitialize   = modpcanbasic.NewProc("CAN_Uninitialize")
	procCAN_Reset          = modpcanbasic.NewProc("CAN_Reset")
	procCAN_GetStatus      = modpcanbasic.NewProc("CAN_GetStatus")
	procCAN_Read           = modpcanbasic.NewProc("CAN_Read")
	procCAN_Write          = modpcanbasic.NewProc("CAN_Write")
	procCAN_FilterMessages = modpcanbasic.NewProc("CAN_FilterMessages")
	procCAN_SetValue       = modpcanbasic.NewProc("CAN_SetValue")
	procCAN_GetValue       = modpcanbasic.NewProc("CAN_GetValue")
	procCAN_GetErrorText   = modpcanbasic.NewProc("CAN_GetErrorText")
)

func initialize(h Handle, btr0btr1 Baudrate, hw HwType, ioport uint32, intr uint16) (status Status) {
	r0, _, _ := syscall.Syscall6(procCAN_Initialize.Addr(), 5, uintptr(h), uintptr(btr0btr1), uintptr(hw), uintptr(ioport), uintptr(intr), 0)
	status = Status(r0)
	return
}

func uninitialize(h Handle) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_Uninitialize.Addr(), 1, uintptr(h), 0, 0)
	status = Status(r0)
	return
}

func reset(h Handle) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_Reset.Addr(), 1, uintptr(h), 0, 0)
	status = Status(r0)
	return
}

func status(h Handle) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_GetStatus.Addr(), 1, uintptr(h), 0, 0)
	status = Status(r0)
	return
}

func readMsg(h Handle, buf *Msg, ts *TimeStamp) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_Read.Addr(), 3, uintptr(h), uintptr(unsafe.Pointer(buf)), uintptr(unsafe.Pointer(ts)))
	status = Status(r0)
	return
}

func writeMsg(h Handle, buf *Msg) (status Status) {
	r0, _, _ := syscall.Syscall(procCAN_Write.Addr(), 2, uintptr(h), uintptr(unsafe.Pointer(buf)), 0)
	status = Status(r0)
	return
}

func filterMsgs(h Handle, fromID uint32, toID uint32, mode Mode) (status Status) {
	r0, _, _ := syscall.Syscall6(procCAN_FilterMessages.Addr(), 4, uintptr(h), uintptr(fromID), uintptr(toID), uintptr(mode), 0, 0)
	status = Status(r0)
	return
}

func setValue(h Handle, p byte, buf uintptr, size uintptr) (s Status) {
	r0, _, _ := syscall.Syscall6(procCAN_SetValue.Addr(), 4, uintptr(h), uintptr(p), uintptr(buf), uintptr(size), 0, 0)
	s = Status(r0)
	return
}

func getValue(h Handle, p byte, buf uintptr, size uintptr) (s Status) {
	r0, _, _ := syscall.Syscall6(procCAN_GetValue.Addr(), 4, uintptr(h), uintptr(p), uintptr(buf), uintptr(size), 0, 0)
	s = Status(r0)
	return
}

func errorText(err Status, lang uint16, buf *byte) (s Status) {
	r0, _, _ := syscall.Syscall(procCAN_GetErrorText.Addr(), 3, uintptr(err), uintptr(lang), uintptr(unsafe.Pointer(buf)))
	s = Status(r0)
	return
}
