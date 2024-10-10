package dev

import "github.com/knieriem/can/timing"

var LPC21xx = &timing.DevSpec{
	TSeg1Max:     16,
	TSeg2Max:     16,
	SJWMax:       32,
	PrescalerMax: 512,

	// EncodeToReg calculates a Bus Timing Register (BTR) value,
	// see UM10114, page 284.
	EncodeToReg: func(t *timing.BitTiming) *timing.RegValue {
		r := uint32((t.TSeg1()-1)&0xF) << 16
		r |= uint32((t.TSeg2()-1)&7) << 20
		r |= uint32(t.Prescaler - 1)
		r |= uint32((t.SJW-1)&3) << 14
		return &timing.RegValue{Reg32: r}
	},
	DecodeReg: func(rv *timing.RegValue) *timing.BitTiming {
		btr := rv.Reg32
		t := new(timing.BitTiming)
		t.Prescaler = int((btr & 0x1FF) + 1)
		t.SJW = int(((btr >> 14) & 3) + 1)

		tseg1 := int(((btr >> 16) & 0xF) + 1)
		t.PropSeg = 1
		t.PhaseSeg1 = tseg1 - 1

		t.PhaseSeg2 = int(((btr >> 20) & 0x7) + 1)
		return t
	},
}
