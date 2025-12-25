package api

const (
	NoneBus Handle = 0x00

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
	BitrateInfoFd                    = 0x19
	BusspeedNominal                  = 0x1A
	BusspeedFd                       = 0x1B
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
	HardResetStatus                  = 0x2E
	LanChanDirection                 = 0x2F
	DeviceGuid                       = 0x30
	BitrateInfoCc                    = 0x31
	BitrateInfoXl                    = 0x32
	BusspeedXl                       = 0x33

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

	TraceFileSingle     = 0x00
	TraceFileSegmented  = 0x01
	TraceFileDate       = 0x02
	TraceFileTime       = 0x04
	TraceFileOverwrite  = 0x80
	TraceFileDataLength = 0x100

	FeatureFdCapable    = 0x01
	FeatureDelayCapable = 0x02
	FeatureIoCapable    = 0x04
	FeatureXlCapable    = 0x08

	ServiceStatusStopped = 0x01
	ServiceStatusRunning = 0x04

	LanDirectionRead      = 0x01
	LanDirectionWrite     = 0x02
	LanDirectionReadWrite = (LanDirectionRead | LanDirectionWrite)

	MaxLengthHardwareName  = 33
	MaxLengthVersionString = 256
	MaxLengthDataXl        = 2048
	MaxValueStdId          = 0x7FF
	MaxValueExtId          = 0x1FFFFFFF
	MaxValuePriorityId     = 0x7FF

	MsgStandard = 0x00
	MsgRtr      = 0x01
	MsgExtended = 0x02
	MsgFd       = 0x04
	MsgBrs      = 0x08
	MsgEsi      = 0x10
	MsgEcho     = 0x20
	MsgErrframe = 0x40
	MsgStatus   = 0x80

	MsgXl                = 0x100
	MsgProtocolException = 0x200
	MsgErrnotification   = 0x400

	LookupDeviceType       = "devicetype"
	LookupDeviceId         = "deviceid"
	LookupControllerNumber = "controllernumber"
	LookupIpAddress        = "ipaddress"
	LookupDeviceGuid       = "deviceguid"

	ModeStandard = MsgStandard
	ModeExtended = MsgExtended
)
