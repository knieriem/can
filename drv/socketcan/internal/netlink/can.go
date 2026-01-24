package netlink

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/jsimonetti/rtnetlink"
	"github.com/knieriem/can"
	"github.com/knieriem/can/timing"
	"github.com/mdlayher/netlink"
)

const (
	ifla_CAN_TDC          = 0x10
	ifla_CAN_CTRLMODE_EXT = 0x11

	ifla_CAN_CTRLMODE_SUPPORTED = 1
)

// CanAttributes contain the attributes read from a CAN network interface.
// Fields that are defined as pointers may be nil if they have not been included
// by the kernel.
//
// See  https://github.com/torvalds/linux/blob/master/drivers/net/can/dev/netlink.c
// for details.
//
// This structure does not provide access to IFLA_CAN_CTRLMODE_EXT,
// which has not been ported to Linux v5.15 yet.
// Also, IFLA_CAN_TDC_* is not supported.
type CanAttributes struct {

	// State contains one of unix.CAN_STATE_* values.
	State uint32

	// CtrlMode may be used to access unix.CAN_CTRLMODE_* flags,
	// e.g. for activating FD mode or to configure termination.
	CtrlMode          unix.CANCtrlMode
	CtrlModeSupported uint32

	RestartMs uint32
	Clock     uint32

	BusErrCounters *unix.CANBusErrorCounters

	// BitTiming defines the bit rate, or, alternatively,
	// bit timing parameters. When writing, either the
	// the bit rate or the timing parameters should be left blank.
	BitTiming      *unix.CANBitTiming
	BitTimingConst *unix.CANBitTimingConst

	DataBitTiming      *unix.CANBitTiming
	DataBitTimingConst *unix.CANBitTimingConst

	BitrateMax uint32

	UnknownTypes []uint16
}

func decodeCanAttributes(data []byte) (*CanAttributes, error) {
	netlAd, err := netlink.NewAttributeDecoder(data)
	if err != nil {
		return nil, err
	}
	ad := new(canAttrDecoder)
	ad.AttributeDecoder = netlAd

	c := new(CanAttributes)
	for ad.Next() {
		switch ad.Type() {
		default:
			c.UnknownTypes = append(c.UnknownTypes, ad.Type())
		case unix.IFLA_CAN_BERR_COUNTER:
			cnt := new(unix.CANBusErrorCounters)
			err := ad.decodeStruct(cnt)
			if err != nil {
				return nil, fmt.Errorf("parsing CtrlMode: %w", err)
			}
			c.BusErrCounters = cnt
		case unix.IFLA_CAN_STATE:
			c.State = ad.Uint32()
		case unix.IFLA_CAN_CTRLMODE:
			err := ad.decodeStruct(&c.CtrlMode)
			if err != nil {
				return nil, fmt.Errorf("parsing CtrlMode: %w", err)
			}
		case ifla_CAN_CTRLMODE_EXT:
			flags, err := ad.decodeCtrlModeExt()
			if err != nil {
				return nil, fmt.Errorf("parsing CtrlModeExt: %w", err)
			}
			c.CtrlModeSupported = flags
		case unix.IFLA_CAN_RESTART_MS:
			c.RestartMs = ad.Uint32()

		case unix.IFLA_CAN_BITTIMING:
			c.BitTiming, err = ad.decodeBitTiming("")

		case unix.IFLA_CAN_BITTIMING_CONST:
			c.BitTimingConst, err = ad.decodeBitTimingConst("")

		case unix.IFLA_CAN_DATA_BITTIMING:
			c.DataBitTiming, err = ad.decodeBitTiming("DATA_")

		case unix.IFLA_CAN_DATA_BITTIMING_CONST:
			c.DataBitTimingConst, err = ad.decodeBitTimingConst("DATA_")

		case unix.IFLA_CAN_CLOCK:
			c.Clock = ad.Uint32()

		case unix.IFLA_CAN_BITRATE_MAX:
			c.BitrateMax = ad.Uint32()
		}
		if err := ad.Err(); err != nil {
			return nil, err
		}
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

type canAttrDecoder struct {
	*netlink.AttributeDecoder
}

func (ad *canAttrDecoder) decodeBitTiming(namePrefix string) (*unix.CANBitTiming, error) {
	var bt unix.CANBitTiming
	err := ad.decodeStruct(&bt)
	if err != nil {
		return nil, fmt.Errorf("parsing CAN_%s: %w", namePrefix, err)
	}
	return &bt, nil
}

func (ad *canAttrDecoder) decodeBitTimingConst(namePrefix string) (*unix.CANBitTimingConst, error) {
	var bt unix.CANBitTimingConst
	err := ad.decodeStruct(&bt)
	if err != nil {
		return nil, fmt.Errorf("parsing CAN_%sBITTIMING_CONST: %w", namePrefix, err)
	}
	return &bt, nil
}

func (ad *canAttrDecoder) decodeStruct(v any) error {
	var err error

	ad.Do(func(data []byte) error {
		err = binary.Read(bytes.NewReader(ad.Bytes()), ad.ByteOrder, v)
		return nil
	})
	return err
}

func (ad *canAttrDecoder) decodeCtrlModeExt() (uint32, error) {
	var mask uint32
	ad.Do(func(b []byte) error {
		xad, _ := netlink.NewAttributeDecoder(b)
		for xad.Next() {
			if xad.Type() == ifla_CAN_CTRLMODE_SUPPORTED {
				mask = xad.Uint32()
				return nil
			}
		}
		return nil
	})
	return mask, ad.Err()
}

func (can *CanAttributes) Controller() *timing.Controller {
	fdCapable := can.CtrlModeSupported&unix.CAN_CTRLMODE_FD != 0
	if can.DataBitTimingConst == nil {
		fdCapable = false
	}
	ctl := new(timing.Controller)
	ctl.Clock = can.Clock
	convertConstraints(&ctl.Nominal, can.BitTimingConst)
	if fdCapable {
		ctl.Data = new(timing.Constraints)
		convertConstraints(ctl.Data, can.BitTimingConst)
	}
	return ctl
}

func convertConstraints(cstr *timing.Constraints, c *unix.CANBitTimingConst) {
	cstr.TSeg1Min = int(c.Tseg1_min)
	cstr.TSeg1Max = int(c.Tseg1_max)
	cstr.TSeg2Min = int(c.Tseg2_min)
	cstr.TSeg2Max = int(c.Tseg2_max)

	cstr.SJWMax = int(c.Sjw_max)

	cstr.PrescalerMin = int(c.Brp_min)
	cstr.PrescalerMax = int(c.Brp_max)
	cstr.PrescalerIncr = int(c.Brp_inc)
}

func (can *canAttrEncoder) UpdateLink(link *Interface) error {
	data, err := can.ae.Encode()
	if err != nil {
		return err
	}
	cur, err := link.get()
	if err != nil {
		return err
	}
	var modified uint32
	if op := cur.Attributes.OperationalState; op == rtnetlink.OperStateUp || op == rtnetlink.OperStateUnknown {
		modified = unix.IFF_UP
	}
	msg := &rtnetlink.LinkMessage{
		Family: link.msgFamily,
		Type:   link.msgType,
		Index:  uint32(link.index),
		Change: modified,
	}
	err = link.conn.Link.Set(msg)
	if err != nil {
		return err
	}
	msg = &rtnetlink.LinkMessage{
		Family: link.msgFamily,
		Type:   link.msgType,
		Index:  uint32(link.index),
		Attributes: &rtnetlink.LinkAttributes{
			Info: &rtnetlink.LinkInfo{
				Kind: "can",
				Data: data,
			},
		},
	}
	return link.conn.Link.Set(msg)
}

type canAttrEncoder struct {
	ae *netlink.AttributeEncoder
}

func NewCANAttrEncoder() *canAttrEncoder {
	can := new(canAttrEncoder)
	can.ae = netlink.NewAttributeEncoder()
	return can
}

func (can *canAttrEncoder) SetConfig(conf *can.Config) {
	can.setBittiming(unix.IFLA_CAN_BITTIMING, &conf.Nominal)
	fd := (conf.FDMode.Valid && conf.FDMode.Value) || conf.Data.Valid
	if fd {
		data := &conf.Nominal
		if conf.Data.Valid {
			data = &conf.Data.Value
		}
		can.setBittiming(unix.IFLA_CAN_DATA_BITTIMING, data)
	}
	can.SetFDMode(fd)
	// TODO: support conf.Termination
}

func (can *canAttrEncoder) SetFDMode(v bool) {
	var m unix.CANCtrlMode
	m.Mask = unix.CAN_CTRLMODE_FD
	if v {
		m.Flags = m.Mask
	}
	can.encodeData(unix.IFLA_CAN_CTRLMODE, m)
}

func (can *canAttrEncoder) setBittiming(t uint16, btc *can.BitTimingConfig) {
	var bt unix.CANBitTiming
	if btc.Tq != 0 {
		bt.Tq = uint32(btc.Tq)
		bt.Prop_seg = uint32(btc.PropSeg)
		bt.Phase_seg1 = uint32(btc.PhaseSeg1)
		bt.Phase_seg2 = uint32(btc.PhaseSeg2)
	} else {
		bt.Bitrate = btc.Bitrate
		bt.Sample_point = uint32(btc.SamplePoint)
	}
	bt.Sjw = uint32(btc.SJW)
	can.encodeData(t, &bt)
}

func (can *canAttrEncoder) encodeData(t uint16, data any) {
	can.ae.Do(t, func() ([]byte, error) {
		var b bytes.Buffer
		err := binary.Write(&b, can.ae.ByteOrder, data)
		if err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	})
}
