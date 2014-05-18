package pcan

import (
	"bytes"
)

type MsgData struct {
	ID    uint32
	RTR   uint8
	ExtID uint8
	Len   uint8
	Data  [8]uint8
}

//sys Open(device string) (fd int, err error) [failretval < 0] = jpcan.pcan_open

//sys SendMsg(fd int, msg *MsgData) (rc int32, err error) [failretval != 0] = jpcan.pcan_send_msg

//sys InitPool(fd int, size uint32) (err error) [failretval != 0] = jpcan.pcan_init_pool

//sys Idvers(fd int, mod *uint32, car *uint32) (err error) [failretval != 0] = jpcan.pcan_idvers

//sys	SetBTR(fd int, btr uint16) (err error)  [failretval != 0] = jpcan.pcan_setbtr

//sys	BusOn(fd int) (err error)  [failretval != 0] = jpcan.pcan_buson
//sys	BusOff(fd int) (err error)  [failretval != 0] = jpcan.pcan_busoff

//sys ConfigTerm(fd int, term uint8) (err error)  [failretval != 0] = jpcan.pcan_config_term

//sys Close(fd int) (err error)  [failretval != 0] = jpcan.pcan_close

//
// Functions implemented in the helper DLL:
//

//sys CreateHelper(fd int) (err error) [failretval != 0] = jpcangohelper.create
//sys MsgAvail(fd int) (n int) = jpcangohelper.msgavail
//sys ReadMsg(fd int, buf *uint8) (err error) [failretval != 0] = jpcangohelper.readmsg
//sys CloseHelper(fd int) (err error) = jpcangohelper.close

type DeviceInfo struct {
	name   [32]byte
	sernum [32]byte
	Flags  uint32
	Type   uint32
	ID     uint32
	LocID  uint32
	desc   [64]byte
	_      uint32
}

func (di *DeviceInfo) Name() string {
	return bytesToString(di.name[:])
}

func (di *DeviceInfo) Desc() string {
	return bytesToString(di.desc[:])
}

func (di *DeviceInfo) SerialNum() string {
	return bytesToString(di.sernum[:])
}

func bytesToString(buf []byte) string {
	i := bytes.IndexByte(buf, 0)
	if i != -1 {
		buf = buf[:i]
	}
	return string(buf)
}

//sys USBDevices(list []DeviceInfo) (n int, err error) [failretval == -1] = jpcangohelper.usbdevices
