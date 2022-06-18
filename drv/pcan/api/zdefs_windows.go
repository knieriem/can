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

	PCIBUS1  Handle = 0x41
	PCIBUS2  Handle = 0x42
	PCIBUS3  Handle = 0x43
	PCIBUS4  Handle = 0x44
	PCIBUS5  Handle = 0x45
	PCIBUS6  Handle = 0x46
	PCIBUS7  Handle = 0x47
	PCIBUS8  Handle = 0x48
	PCIBUS9  Handle = 0x409
	PCIBUS10 Handle = 0x40A
	PCIBUS11 Handle = 0x40B
	PCIBUS12 Handle = 0x40C
	PCIBUS13 Handle = 0x40D
	PCIBUS14 Handle = 0x40E
	PCIBUS15 Handle = 0x40F
	PCIBUS16 Handle = 0x410

	USBBUS1  Handle = 0x51
	USBBUS2  Handle = 0x52
	USBBUS3  Handle = 0x53
	USBBUS4  Handle = 0x54
	USBBUS5  Handle = 0x55
	USBBUS6  Handle = 0x56
	USBBUS7  Handle = 0x57
	USBBUS8  Handle = 0x58
	USBBUS9  Handle = 0x509
	USBBUS10 Handle = 0x50A
	USBBUS11 Handle = 0x50B
	USBBUS12 Handle = 0x50C
	USBBUS13 Handle = 0x50D
	USBBUS14 Handle = 0x50E
	USBBUS15 Handle = 0x50F
	USBBUS16 Handle = 0x510

	PCCBUS1 Handle = 0x61
	PCCBUS2 Handle = 0x62

	LANBUS1  Handle = 0x801
	LANBUS2  Handle = 0x802
	LANBUS3  Handle = 0x803
	LANBUS4  Handle = 0x804
	LANBUS5  Handle = 0x805
	LANBUS6  Handle = 0x806
	LANBUS7  Handle = 0x807
	LANBUS8  Handle = 0x808
	LANBUS9  Handle = 0x809
	LANBUS10 Handle = 0x80A
	LANBUS11 Handle = 0x80B
	LANBUS12 Handle = 0x80C
	LANBUS13 Handle = 0x80D
	LANBUS14 Handle = 0x80E
	LANBUS15 Handle = 0x80F
	LANBUS16 Handle = 0x810

	OK              Status = 0x00000
	ErrXMTFULL      Status = 0x00001
	ErrOVERRUN      Status = 0x00002
	ErrBUSLIGHT     Status = 0x00004
	ErrBUSHEAVY     Status = 0x00008
	ErrBUSWARNING   Status = ErrBUSHEAVY
	ErrBUSPASSIVE   Status = 0x40000
	ErrBUSOFF       Status = 0x00010
	ErrANYBUSERR    Status = (ErrBUSWARNING | ErrBUSLIGHT | ErrBUSHEAVY | ErrBUSOFF | ErrBUSPASSIVE)
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
	ErrILLMODE      Status = 0x80000
	ErrCAUTION      Status = 0x2000000
	ErrINITIALIZE   Status = 0x4000000
	ErrILLOPERATION Status = 0x8000000

	DeviceId                         = 0x01
	FiveVoltsPower         BoolPar   = 0x02
	ReceiveEvent           HandlePar = 0x03
	MsgFilter                        = 0x04
	ApiVersion             StringPar = 0x05
	ChanVersion            StringPar = 0x06
	BusoffAutoreset        BoolPar   = 0x07
	ListenOnly             BoolPar   = 0x08
	LogLocation            StringPar = 0x09
	LogStatus              BoolPar   = 0x0A
	LogConfigure           IntPar    = 0x0B
	LogText                StringPar = 0x0C
	ChanCondition          IntPar    = 0x0D
	HardwareName           StringPar = 0x0E
	ReceiveStatus          BoolPar   = 0x0F
	ControllerNumber       IntPar    = 0x10
	TraceLocation                    = 0x11
	TraceStatus                      = 0x12
	TraceSize                        = 0x13
	TraceConfigure                   = 0x14
	ChanIdentifying                  = 0x15
	ChanFeatures                     = 0x16
	BitrateAdapting                  = 0x17
	BitrateInfo                      = 0x18
	BitrateInfoFd                    = 0x19
	BusspeedNominal                  = 0x1A
	BusspeedData                     = 0x1B
	IpAddress                        = 0x1C
	LanServiceStatus                 = 0x1D
	AllowStatusFrames                = 0x1E
	AllowRtrFrames                   = 0x1F
	AllowErrframes                   = 0x20
	InterframeDelay                  = 0x21
	AcceptanceFilter11bit            = 0x22
	AcceptanceFilter29bit            = 0x23
	IoDigitalConfiguration           = 0x24
	IoDigitalValue                   = 0x25
	IoDigitalSet                     = 0x26
	IoDigitalClear                   = 0x27
	IoAnalogValue                    = 0x28
	FirmwareVersion                  = 0x29
	AttachedChannelsCount            = 0x2A
	AttachedChannels                 = 0x2B
	AllowEchoFrames                  = 0x2C
	DevicePartNumber                 = 0x2D

	ParamOff        = 0x00
	ParamOn         = 0x01
	FilterClose     = 0x00
	FilterOpen      = 0x01
	FilterCustom    = 0x02
	ChanUnavailable = 0x00
	ChanAvailable   = 0x01
	ChanOccupied    = 0x02
	ChanPcanview    = (ChanAvailable | ChanOccupied)

	LogFnDefault = 0x00
	LogFnEntry   = 0x01
	LogFnParams  = 0x02
	LogFnLeave   = 0x04
	LogFnWrite   = 0x08
	LogFnRead    = 0x10
	LogFnAll     = 0xFFFF

	TraceFileSingle    = 0x00
	TraceFileSegmented = 0x01
	TraceFileDate      = 0x02
	TraceFileTime      = 0x04
	TraceFileOverwrite = 0x80

	FeatureFdCapable    = 0x01
	FeatureDelayCapable = 0x02
	FeatureIoCapable    = 0x04

	ServiceStatusStopped = 0x01
	ServiceStatusRunning = 0x04

	MaxLengthHardwareName  = 33
	MaxLengthVersionString = 256

	MsgStandard = 0x00
	MsgRtr      = 0x01
	MsgExtended = 0x02
	MsgFd       = 0x04
	MsgBrs      = 0x08
	MsgEsi      = 0x10
	MsgEcho     = 0x20
	MsgErrframe = 0x40
	MsgStatus   = 0x80

	LookupDeviceType       = "devicetype"
	LookupDeviceId         = "deviceid"
	LookupControllerNumber = "controllernumber"
	LookupIpAddress        = "ipaddress"

	ModeStandard = MsgStandard
	ModeExtended = MsgExtended

	BR_CLOCK       = "f_clock"
	BR_CLOCK_MHZ   = "f_clock_mhz"
	BR_NOM_BRP     = "nom_brp"
	BR_NOM_TSEG1   = "nom_tseg1"
	BR_NOM_TSEG2   = "nom_tseg2"
	BR_NOM_SJW     = "nom_sjw"
	BR_NOM_SAMPLE  = "nom_sam"
	BR_DATA_BRP    = "data_brp"
	BR_DATA_TSEG1  = "data_tseg1"
	BR_DATA_TSEG2  = "data_tseg2"
	BR_DATA_SJW    = "data_sjw"
	BR_DATA_SAMPLE = "data_ssp_offset"

	TypeISA         HwType = 0x01
	TypeISA_SJA     HwType = 0x09
	TypeISA_PHYTEC  HwType = 0x04
	TypeDNG         HwType = 0x02
	TypeDNG_EPP     HwType = 0x03
	TypeDNG_SJA     HwType = 0x05
	TypeDNG_SJA_EPP HwType = 0x06
)
