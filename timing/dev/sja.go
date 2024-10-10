package dev

import "github.com/knieriem/can/timing"

// SJA1000 defines register constraints for the SJA1000;
// see https://www.nxp.com/docs/en/data-sheet/SJA1000.pdf#page=50
var SJA1000 = &timing.DevSpec{
	TSeg1Max:     16,
	TSeg2Max:     8,
	SJWMax:       4,
	PrescalerMax: 32,

	EncodeToReg: func(t *timing.BitTiming) *timing.RegValue {
		btr0 := uint8((t.SJW-1)<<6 | (t.Prescaler - 1))
		btr1 := uint8(((t.TSeg2() - 1) << 4) | (t.TSeg1() - 1))
		return &timing.RegValue{Reg8: []byte{btr0, btr1}}
	},
	DecodeReg: func(rv *timing.RegValue) *timing.BitTiming {
		regs := rv.Reg8
		if len(regs) != 2 {
			return nil
		}
		btr0 := regs[0]
		btr1 := regs[1]

		t := new(timing.BitTiming)
		t.SJW = ((int(btr0) >> 6) & 3) + 1
		t.Prescaler = (int(btr0) & 0x3F) + 1

		t.PhaseSeg2 = ((int(btr1) >> 4) & 7) + 1
		tseg1 := int((btr1 & 0xF) + 1)

		d := &timing.DevSpec{TSeg1Max: 16}
		prop, ps1 := d.SplitTSeg1(tseg1, int(t.PhaseSeg2)-1)
		t.PropSeg = int(prop)
		t.PhaseSeg1 = int(ps1)
		return t
	},
}
