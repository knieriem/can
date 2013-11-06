package api

const (
	NoneBus Handle = 0x00

	ISABUS1 Handle = 0x21
	ISABUS2 Handle = 0x22
	ISABUS3 Handle = 0x23
	ISABUS4 Handle = 0x24
	ISABUS5 Handle = 0x25
	ISABUS6 Handle = 0x26
	ISABUS7 Handle = 0x27
	ISABUS8 Handle = 0x28

	DNGBUS1 Handle = 0x31

	PCIBUS1 Handle = 0x41
	PCIBUS2 Handle = 0x42
	PCIBUS3 Handle = 0x43
	PCIBUS4 Handle = 0x44
	PCIBUS5 Handle = 0x45
	PCIBUS6 Handle = 0x46
	PCIBUS7 Handle = 0x47
	PCIBUS8 Handle = 0x48

	USBBUS1 Handle = 0x51
	USBBUS2 Handle = 0x52
	USBBUS3 Handle = 0x53
	USBBUS4 Handle = 0x54
	USBBUS5 Handle = 0x55
	USBBUS6 Handle = 0x56
	USBBUS7 Handle = 0x57
	USBBUS8 Handle = 0x58

	PCCBUS1 Handle = 0x61
	PCCBUS2 Handle = 0x62

	OK              Status = 0x00000
	ErrXMTFULL      Status = 0x00001
	ErrOVERRUN      Status = 0x00002
	ErrBUSLIGHT     Status = 0x00004
	ErrBUSHEAVY     Status = 0x00008
	ErrBUSOFF       Status = 0x00010
	ErrANYBUSERR    Status = (ErrBUSLIGHT | ErrBUSHEAVY | ErrBUSOFF)
	ErrQRCVEMPTY    Status = 0x00020
	ErrQOVERRUN     Status = 0x00040
	ErrQXMTFULL     Status = 0x00080
	ErrREGTEST      Status = 0x00100
	ErrNODRIVER     Status = 0x00200
	ErrHWINUSE      Status = 0x00400
	ErrNETINUSE     Status = 0x00800
	ErrILLHW        Status = 0x01400
	ErrILLNET       Status = 0x01800
	ErrILLCLIENT    Status = 0x01C00
	ErrILLHANDLE    Status = (ErrILLHW | ErrILLNET | ErrILLCLIENT)
	ErrRESOURCE     Status = 0x02000
	ErrILLPARAMTYPE Status = 0x04000
	ErrILLPARAMVAL  Status = 0x08000
	ErrUNKNOWN      Status = 0x10000
	ErrILLDATA      Status = 0x20000
	ErrINITIALIZE   Status = 0x40000

	DeviceNumber     IntPar    = 0x01
	FiveVoltsPower   BoolPar   = 0x02
	ReceiveEvent     HandlePar = 0x03
	MsgFilter                  = 0x04
	ApiVersion       StringPar = 0x05
	ChanVersion      StringPar = 0x06
	BusoffAutoreset  BoolPar   = 0x07
	ListenOnly       BoolPar   = 0x08
	LogLocation      StringPar = 0x09
	LogStatus        BoolPar   = 0x0A
	LogConfigure     IntPar    = 0x0B
	LogText          StringPar = 0x0C
	ChanCondition    IntPar    = 0x0D
	HardwareName     StringPar = 0x0E
	ReceiveStatus    BoolPar   = 0x0F
	ControllerNumber IntPar    = 0x10

	ParamOff        = 0x00
	ParamOn         = 0x01
	FilterClose     = 0x00
	FilterOpen      = 0x01
	FilterCustom    = 0x02
	ChanUnavailable = 0x00
	ChanAvailable   = 0x01
	ChanOccupied    = 0x02

	LogFnDefault = 0x00
	LogFnEntry   = 0x01
	LogFnParams  = 0x02
	LogFnLeave   = 0x04
	LogFnWrite   = 0x08
	LogFnRead    = 0x10
	LogFnAll     = 0xFFFF

	MsgStandard = 0x00
	MsgRtr      = 0x01
	MsgExtended = 0x02
	MsgStatus   = 0x80

	ModeStandard = MsgStandard
	ModeExtended = MsgExtended

	TypeISA         HwType = 0x01
	TypeISA_SJA     HwType = 0x09
	TypeISA_PHYTEC  HwType = 0x04
	TypeDNG         HwType = 0x02
	TypeDNG_EPP     HwType = 0x03
	TypeDNG_SJA     HwType = 0x05
	TypeDNG_SJA_EPP HwType = 0x06
)
