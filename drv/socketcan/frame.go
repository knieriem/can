//go:build linux

package socketcan

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/socketcan/internal/linux"
)

const (
	// linux CAN frame
	idLen       = 4
	flagsOffset = 5
	lenOffset   = idLen
	dataOffset  = 8
)

// so far we only support littleEndian architectures
var native = binary.LittleEndian

type frame struct {
	b [linux.CANFD_MTU]byte
}

func (f *frame) id() uint32 {
	return native.Uint32(f.b[0:idLen])
}

func (f *frame) setid(id uint32) {
	native.PutUint32(f.b[0:idLen], id)
}

func (f *frame) len() int {
	return int(f.b[lenOffset])
}

func (f *frame) setLen(n int) {
	f.b[lenOffset] = byte(n)
}

func (f *frame) setFlags(v byte) {
	f.b[flagsOffset] = byte(v)
}

func (f *frame) data() []byte {
	return f.b[dataOffset:]
}

func (f *frame) encode(msg *can.Msg, mtu int) (nw int, err error) {
	nw = linux.CAN_MTU

	data := msg.Data()
	n := len(data)
	if n > mtu-dataOffset {
		return 0, ErrMTUExceeded
	}
	_, needsFD, err := can.VerifyDataLenFD(n)
	if err != nil {
		return 0, err
	}
	f.setLen(n)
	copy(f.data(), data)

	id := msg.Id
	if msg.Test(can.RTRMsg) {
		if n != 0 {
			return 0, can.ErrInvalidMsgLen
		}
		id |= linux.CAN_RTR_FLAG
	}
	if msg.ExtFrame() {
		id |= linux.CAN_EFF_FLAG
	}
	f.setid(id)

	if needsFD || msg.Test(can.ForceFD) {
		nw = len(f.b)
	}
	if msg.Flags.Test(can.FDSwitchBitrate) {
		f.setFlags(linux.CANFD_BRS)
	}
	return nw, nil
}

func (f *frame) decode(msg *can.Msg) error {
	id := f.id()
	msg.Reset()
	if id&linux.CAN_ERR_FLAG != 0 {
		// error frame
		msg.Flags |= can.StatusMsg

		errClass := id & linux.CAN_ERR_MASK
		if errClass&linux.CAN_ERR_BUSOFF != 0 {
			msg.Flags |= can.BusOff
		}
		if errClass&linux.CAN_ERR_CRTL != 0 {
			if f.data()[1]&linux.CAN_ERR_CRTL_RX_OVERFLOW != 0 {
				msg.Flags |= can.ReceiveBufferOverflow
			}
		}
		return nil
	}

	idMask := uint32(linux.CAN_SFF_MASK)
	if id&linux.CAN_EFF_FLAG != 0 {
		/* extended frame */
		msg.Flags = can.ExtFrame
		idMask = linux.CAN_EFF_MASK
	}
	msg.Id = id & idMask
	if id&linux.CAN_RTR_FLAG != 0 {
		msg.Flags = can.RTRMsg
		return nil
	}
	data := msg.Data()
	n := f.len()
	if cap(data) < n {
		return can.ErrMsgCapExceeded
	}
	data = data[:n]
	msg.SetData(data)
	copy(data, f.data())
	return nil
}

func (f *frame) readFromN(r io.Reader, n int) error {
	n, err := r.Read(f.b[:n])
	if err != nil {
		return err
	}

	// check for frame integrity
	if n < dataOffset || dataOffset+f.len() > n {
		return fmt.Errorf("unexpected short read: %d bytes", n)
	}
	return nil
}

var ErrMTUExceeded = errors.New("frame length exceeds MTU")
