package netlink

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/mdlayher/netlink"
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
	CtrlMode unix.CANCtrlMode

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
