package timing

import (
	"fmt"
	"time"
)

// Bittiming defines the CAN bit timing values.
// The length of a time quantum, as mentioned below,
// is the oscillator period multiplied with the prescaler value
// (and, depending on the device, the value of an extra device-internal divider).
type BitTiming struct {
	// Prescaler defines the value of a device's programmable prescaler.
	Prescaler int

	// PropSeg defines the length of the propagation time segment, in time quanta.
	PropSeg int

	// PhaseSeg1 defines the length of the phase buffer segment 1, in time quanta.
	PhaseSeg1 int

	// PhaseSeg2 defines the length of the phase buffer segment 2, in time quanta.
	PhaseSeg2 int

	// SJW defines the resynchronization jump width, in time quanta.
	SJW int
}

// DevSpec defines device specific limits and properties.
// CAN bit timing configuration registers of a specific device
// allow values for the time segments, the prescaler, and sjw
// within defined ranges only, dependent on the number of bits
// available for each value.
type DevSpec struct {
	PropSegMax int
	TSeg1Min   int
	TSeg1Max   int
	TSeg2Min   int
	TSeg2Max   int

	SJWMax int

	PrescalerMin  int
	PrescalerMax  int
	PrescalerIncr int

	// FOscDiv allows to specify an extra divider. In some devices, like MCP2515,
	// fOsc is divided by two, before being fed to the prescaler; in this case,
	// a value of 2 should be assigned to this field.
	FOscDiv int

	EncodeToReg func(*BitTiming) *RegValue
	DecodeReg   func(*RegValue) *BitTiming
}

// DevSpecFD contains the device specific limits for the
// bit timings of the nominal and data bitrates.
type DevSpecFD struct {
	Nominal DevSpec
	Data    DevSpec
}

// RegValue contains the bit timing value encoded into
// one or more registers. Only one field is used,
// depending on the specific device.
type RegValue struct {
	Reg8  []byte
	Reg16 uint16
	Reg32 uint32
}

func (r *RegValue) String() string {
	if len(r.Reg8) != 0 {
		return fmt.Sprintf("{% x}", r.Reg8)
	}
	if r.Reg16 != 0 {
		return fmt.Sprintf("0x%04x", r.Reg16)
	}
	if r.Reg32 != 0 {
		return fmt.Sprintf("0x%08x", r.Reg32)
	}
	return "{}"
}

// NqMax returns the maxmimum number of time quanta that
// can be used on a specific device.
func (dev *DevSpec) NqMax() int {
	return 1 + dev.TSeg1Max + dev.TSeg2Max
}

// TSeg1 returns the length of the time segment 1 in time quanta.
func (t *BitTiming) TSeg1() int {
	return t.PropSeg + t.PhaseSeg1
}

// TSeg2 returns the length of the time segment 2 in time quanta.
func (t *BitTiming) TSeg2() int {
	return t.PhaseSeg2
}

// The value of a time quantum. Note that if a device applies, in addition to the
// prescaler, an extra divider to fOsc, the argument to Tq must be divided by
// that value. For instance, a MCP2515 contains an internal division by two,
// so in case of a 16 MHz oscillator the argument fOsc should be 8 MHz.
func (t *BitTiming) Tq(fOsc uint32) time.Duration {
	return time.Second * time.Duration(t.Prescaler) / time.Duration(fOsc)
}

// Number of time quanta.
func (t *BitTiming) Nq() int {
	return 1 + t.TSeg1() + t.TSeg2()
}

// Bitrate calculates the resulting CAN bitrate for the given fOsc.
// See the comment for the .Tq method for how to reduce fOsc in case
// of an extra device internal divider.
func (t *BitTiming) Bitrate(fOsc uint32) uint32 {
	return fOsc / uint32(t.Prescaler) / uint32(t.Nq())
}

// SamplePoint returns the sample point defined by the BitTiming values.
func (t *BitTiming) SamplePoint() SamplePoint {
	return calcSamplePoint(t.TSeg1(), t.Nq())
}
