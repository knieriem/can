//go:build linux

package socketcan

import (
	"encoding/binary"
	"errors"

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

func (f *frame) flags() byte {
	return f.b[flagsOffset]
}

func (f *frame) setFlags(v byte) {
	f.b[flagsOffset] = byte(v)
}

func (f *frame) data() []byte {
	data := f.b[dataOffset:]
	if n := f.len(); n < len(data) {
		data = data[:n]
	}
	return data
}

func (f *frame) encode(msg *can.Msg, mtu int) (nw int, err error) {
	nw = linux.CAN_MTU

	clear(f.b[:])
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
	if msg.Flags&can.RTRMsg != 0 {
		if n != 0 {
			return 0, can.ErrInvalidMsgLen
		}
		id |= linux.CAN_RTR_FLAG
	}
	if msg.ExtFrame() {
		id |= linux.CAN_EFF_FLAG
	}
	f.setid(id)

	if msg.Flags&can.BRS != 0 {
		f.setFlags(linux.CANFD_BRS)
		needsFD = true
	}
	if needsFD || msg.Flags&can.FDF != 0 {
		nw = len(f.b)
	}
	return nw, nil
}

// decode stores the frame in msg. For an error frame, it stores a
// status message, and keeps state, the last known bus state, current.
func (f *frame) decode(msg *can.Msg, nRead int, state *can.BusState) {
	id := f.id()
	msg.Reset()
	if id&linux.CAN_ERR_FLAG != 0 {
		msg.SetStatus(f.decodeStatus(state))
		return
	}

	idMask := uint32(linux.CAN_SFF_MASK)
	if id&linux.CAN_EFF_FLAG != 0 {
		/* extended frame */
		msg.Flags |= can.ExtFrame
		idMask = linux.CAN_EFF_MASK
	}
	msg.Id = id & idMask
	flags := f.flags()

	if isFD := nRead == linux.CANFD_MTU || flags&linux.CANFD_FDF != 0; isFD {
		msg.Flags |= can.FDF
		if flags&linux.CANFD_BRS != 0 {
			msg.Flags |= can.BRS
		}
		if flags&linux.CANFD_ESI != 0 {
			msg.Flags |= can.ESI
		}
	} else if id&linux.CAN_RTR_FLAG != 0 {
		msg.Flags |= can.RTRMsg
		return
	}
	msg.SetData(f.data())
}

// Values from linux/can/error.h that package linux lacks.
const (
	errCrtlActive    = 0x40  // CAN_ERR_CRTL_ACTIVE, in data[1]
	errProtLocAck    = 0x19  // CAN_ERR_PROT_LOC_ACK, in data[3]
	errProtLocAckDel = 0x1B  // CAN_ERR_PROT_LOC_ACK_DEL, in data[3]
	errCnt           = 0x200 // CAN_ERR_CNT: data[6] and data[7] hold the error counters
)

// decodeStatus translates an error frame. Controllers report a missing
// acknowledge differently: as CAN_ERR_ACK, or as a protocol error in
// the ACK slot or delimiter. Both become NoAck.
func (f *frame) decodeStatus(state *can.BusState) can.Status {
	id := f.id() & linux.CAN_ERR_MASK
	d := f.b[dataOffset : dataOffset+8]

	var ev can.Events
	if id&linux.CAN_ERR_ACK != 0 {
		ev |= can.NoAck
	}
	if id&linux.CAN_ERR_PROT != 0 {
		if d[3] == errProtLocAck || d[3] == errProtLocAckDel {
			ev |= can.NoAck
		} else {
			ev |= can.BusError
		}
	}
	if id&linux.CAN_ERR_BUSERROR != 0 {
		ev |= can.BusError
	}
	if id&linux.CAN_ERR_TX_TIMEOUT != 0 {
		ev |= can.TxTimeout
	}
	if id&linux.CAN_ERR_CRTL != 0 {
		c := d[1]
		if c&linux.CAN_ERR_CRTL_RX_OVERFLOW != 0 {
			ev |= can.RxOverflow
		}
		switch {
		case c&(linux.CAN_ERR_CRTL_RX_PASSIVE|linux.CAN_ERR_CRTL_TX_PASSIVE) != 0:
			*state = can.ErrorPassive
		case c&(linux.CAN_ERR_CRTL_RX_WARNING|linux.CAN_ERR_CRTL_TX_WARNING) != 0:
			*state = can.ErrorWarning
		case c&errCrtlActive != 0:
			*state = can.ErrorActive
		}
	}
	if id&linux.CAN_ERR_BUSOFF != 0 {
		*state = can.BusOff
	}
	if id&linux.CAN_ERR_RESTARTED != 0 {
		ev |= can.Restarted
		*state = can.ErrorActive
	}

	st := can.Status{State: *state, Events: ev}
	if id&errCnt != 0 {
		st.TxErr, st.RxErr, st.Counters = d[6], d[7], true
	}
	return st
}

var ErrMTUExceeded = errors.New("frame length exceeds MTU")
