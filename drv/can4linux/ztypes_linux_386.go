// Created by cgo -godefs - DO NOT EDIT
// cgo -godefs types_linux.go

package can4linux

type timeval struct {
	Sec  int32
	Usec int32
}
type msg struct {
	Flags     int32
	Pad_cgo_0 int32
	Id        uint32
	Tstamp    timeval
	Length    int16
	Data      [8]uint8
	Pad_cgo_1 [2]byte
}

type ioctlCmdArg struct {
	Cmd   int32
	Error int32
	Ret   uint32
}
type ioctlConfArg struct {
	Name int32
	Arg1 uint32
	Arg2 int32
	Err  int32
	Ret  uint32
}
type ioctlStatusArg struct {
	Baud            uint32
	StatusReg       uint32
	ErrWarningLimit uint32
	NumRxErrors     uint32
	NumTxErrors     uint32
	ErrCodeReg      uint32
	RxBuf           bufStatus
	TxBuf           bufStatus
	Ret             uint32
	ControllerType  uint32
}
type bufStatus struct {
	Size uint32
	Used uint32
}
