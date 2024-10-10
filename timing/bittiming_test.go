package timing_test

import (
	"slices"
	"testing"
	"time"

	"github.com/knieriem/can/timing"
	"github.com/knieriem/can/timing/dev"
)

type device struct {
	fOsc uint32
	spec *timing.DevSpec
}

type test struct {
	dev     *device
	fOsc    uint32
	bitrate uint32
	sp      timing.SamplePoint
	sjw     int
	opts    []timing.CalcOption

	// expected values
	bt   *timing.BitTiming
	regs *timing.RegValue
	tq   time.Duration

	cmpTSeg1 bool
}

// Most of these tests stem from configurations used during previous years.
var tests = []test{
	{
		// example from: https: //ww1.microchip.com/downloads/en/DeviceDoc/MCP2515-Stand-Alone-CAN-Controller-with-SPI-20001801J.pdf#page=43
		dev:     &mcp2515,
		fOsc:    20e6,
		bitrate: 125e3,
		sp:      625,
		opts: []timing.CalcOption{
			timing.PreferLowerPrescaler(),
		},
		bt: &timing.BitTiming{
			Prescaler: 5,
			PropSeg:   2,
			PhaseSeg1: 7,
			PhaseSeg2: 6,
			SJW:       1,
		},
		cmpTSeg1: true,
	}, {
		dev:     &mcp2515,
		bitrate: 500e3,
		sp:      875,
		regs: &timing.RegValue{
			Reg8: []uint8{0, 0xb5, 1},
		},
	}, {
		dev:     &mcp2515,
		bitrate: 1000e3,
		sp:      875,
		sjw:     3,
		regs: &timing.RegValue{
			Reg8: []uint8{0x40, 0x91, 1},
		},
	}, {
		dev:     &mcp2518fd,
		bitrate: 500e3,
		opts: []timing.CalcOption{
			timing.PreferLowerPrescaler(),
		},
		bt: &timing.BitTiming{
			Prescaler: 1,
			PropSeg:   34,
			PhaseSeg1: 35,
			PhaseSeg2: 10,
			SJW:       1,
		},
		tq: 25 * time.Nanosecond,
	}, {
		dev:     &mcp2518fdData,
		bitrate: 1e6,
		sp:      750,
		opts: []timing.CalcOption{
			timing.PreferLowerPrescaler(),
		},
		bt: &timing.BitTiming{
			Prescaler: 1,
			PropSeg:   14,
			PhaseSeg1: 15,
			PhaseSeg2: 10,
			SJW:       1,
		},
	}, {
		dev:     &lpc21xx,
		bitrate: 250e3,
		regs:    &timing.RegValue{Reg32: 0x1b0003},
	}, {
		dev:     &lpc21xx,
		bitrate: 500e3,
		regs:    &timing.RegValue{Reg32: 0x1b0001},
	}, {
		dev:     &lpc21xx,
		bitrate: 1e6,
		regs:    &timing.RegValue{Reg32: 0x1b0000},
	},
}

var mcp2515 = device{
	fOsc: 16e6,
	spec: dev.MCP2515,
}

var mcp2518fd = device{
	fOsc: 40e6,
	spec: &dev.MCP2518FD.Nominal,
}

var mcp2518fdData = device{
	fOsc: 40e6,
	spec: &dev.MCP2518FD.Data,
}

var lpc21xx = device{
	fOsc: 15e6,
	spec: dev.LPC21xx,
}

func TestCalc(t *testing.T) {
	for i := range tests {
		test := &tests[i]
		dev := test.dev

		fOsc := dev.fOsc
		if f := test.fOsc; f != 0 {
			fOsc = f
		}
		bt, err := timing.CalcBitTiming(fOsc, test.bitrate, test.sp, test.sjw, dev.spec, test.opts...)
		if err != nil {
			t.Error(err)
			continue
		}
		if test.bt != nil {
			if !bitTimingsEqual(test.bt, bt, test.cmpTSeg1) {
				t.Errorf("test %d: timing mismatch %#v != %#v", i, test.bt, bt)
			}
		}
		if x := test.regs; x != nil {
			r := dev.spec.EncodeToReg(bt)
			if !regValueEquals(r, x) {
				t.Errorf("test %d: reg mismatch: %v != %v", i, r, x)
			}
		}
		if xTq := test.tq; xTq != 0 {
			f := fOsc
			if div := dev.spec.FOscDiv; div != 0 {
				f = f / uint32(div)
			}
			if tq := bt.Tq(f); tq != xTq {
				t.Errorf("test %d: tq mismatch: %v != %v", i, tq, xTq)
			}
		}
	}
}

func bitTimingsEqual(bt, bt2 *timing.BitTiming, cmpTReg1 bool) bool {
	if cmpTReg1 {
		if bt.TSeg1() != bt2.TSeg1() {
			return false
		}
	} else {
		if bt.PropSeg != bt2.PropSeg {
			return false
		}
		if bt.PhaseSeg1 != bt2.PhaseSeg1 {
			return false
		}
	}
	if bt.Prescaler != bt2.Prescaler {
		return false
	}
	if bt.SJW != bt2.SJW {
		return false
	}
	return true
}

func regValueEquals(r, r2 *timing.RegValue) bool {
	if len(r.Reg8) != 0 {
		return slices.Compare(r.Reg8, r2.Reg8) == 0
	}
	if r.Reg16 != 0 {
		return r.Reg16 == r2.Reg16
	}
	if r.Reg32 != 0 {
		return r.Reg32 == r2.Reg32
	}
	return false
}
