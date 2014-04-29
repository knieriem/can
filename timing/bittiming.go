package timing

import (
	"time"
)

type BitTiming struct {
	Clock     uint32
	Prescaler uint

	PropSeg   uint
	PhaseSeg1 uint
	PhaseSeg2 uint
	SJW       uint

	TripleSampling bool
}

// The value of a time quantum.
func (t *BitTiming) Tq() (tq time.Duration) {
	tq = time.Second * time.Duration(t.Prescaler) / time.Duration(t.Clock)
	return
}

// Number of time quanta.
func (t *BitTiming) Nq() (nq uint) {
	nq = 1 + t.PropSeg + t.PhaseSeg1 + t.PhaseSeg2
	return
}

// The bit rate.
func (t *BitTiming) Rate() (r uint32) {
	r = t.Clock / uint32(t.Prescaler) / uint32(t.Nq())
	return
}

// The location of the sample point as a fraction between 0 and 1.
func (t *BitTiming) SamplePointLoc() (sp float64) {
	sp = float64(1+t.PropSeg+t.PhaseSeg1) / float64(t.Nq())
	return
}

func (t *BitTiming) BTR0() (b uint) {
	b = (t.SJW-1)*64 + t.Prescaler - 1
	return
}

func (t *BitTiming) BTR1() (b uint) {
	b = (t.PhaseSeg2-1)*16 + t.PhaseSeg1 + t.PropSeg - 1
	if t.TripleSampling {
		b |= 1 << 7
	}
	return
}

func (t *BitTiming) BTR0BTR1() (b uint) {
	b = t.BTR0()<<8 | t.BTR1()
	return
}
