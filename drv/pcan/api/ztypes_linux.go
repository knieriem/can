// Created by cgo -godefs - DO NOT EDIT
// cgo -godefs ,,lintypes.go

package api

type Init struct {
	WBTR0BTR1    uint16
	UcCANMsgType uint8
	UcListenOnly uint8
}
type Msg struct {
	ID        uint32
	MSGTYPE   uint8
	LEN       uint8
	DATA      [8]uint8
	Pad_cgo_0 [2]byte
}
type RMsg struct {
	Msg       Msg
	DwTime    uint32
	WUsec     uint16
	Pad_cgo_0 [2]byte
}
type statusPar struct {
	WErrorFlag uint16
	Pad_cgo_0  [2]byte
	NLastError int32
}
type Diag struct {
	WType           uint16
	Pad_cgo_0       [2]byte
	DwBase          uint32
	WIrqLevel       uint16
	Pad_cgo_1       [2]byte
	DwReadCounter   uint32
	DwWriteCounter  uint32
	DwIRQcounter    uint32
	DwErrorCounter  uint32
	WErrorFlag      uint16
	Pad_cgo_2       [2]byte
	NLastError      int32
	NOpenPaths      int32
	SzVersionString [64]uint8
}
type Btr0Btr1 struct {
	DwBitRate uint32
	WBTR0BTR1 uint16
	Pad_cgo_0 [2]byte
}
type ExtStatus struct {
	WErrorFlag     uint16
	Pad_cgo_0      [2]byte
	NLastError     int32
	NPendingReads  int32
	NPendingWrites int32
}
type MsgFilter struct {
	FromID    uint32
	ToID      uint32
	MSGTYPE   uint8
	Pad_cgo_0 [3]byte
}
type Params struct {
	NSubFunction int32
	Func         [4]byte
}

const (
	ioctlINIT           = 0xc0047a80
	ioctlWRITE_MSG      = 0x40107a81
	ioctlREAD_MSG       = 0x80187a82
	ioctlGET_STATUS     = 0x80087a83
	ioctlDIAG           = 0x80687a84
	ioctlBTR0BTR1       = 0xc0087a85
	ioctlGET_EXT_STATUS = 0x80107a86
	ioctlMSG_FILTER     = 0x400c7a87
	ioctlEXTRA_PARAMS   = 0xc0087a88

	MsgStatus   = 0x80
	MsgExtended = 0x2
	MsgRtr      = 0x1
	MsgStd      = 0x0

	OK              Status = 0x0
	ErrXMTFULL      Status = 0x1
	ErrOVERRUN      Status = 0x2
	ErrBUSLIGHT     Status = 0x4
	ErrBUSHEAVY     Status = 0x8
	ErrBUSOFF       Status = 0x10
	ErrQRCVEMPTY    Status = 0x20
	ErrQOVERRUN     Status = 0x40
	ErrQXMTFULL     Status = 0x80
	ErrREGTEST      Status = 0x100
	ErrNOVXD        Status = 0x200
	ErrRESOURCE     Status = 0x2000
	ErrILLPARAMTYPE Status = 0x4000
	ErrILLPARAMVAL  Status = 0x8000
)
