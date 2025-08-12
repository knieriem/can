package timing

import "errors"

const (
	tq      = 1
	syncSeg = 1 * tq
)

// SamplePoint defines the position within a bit, where the
// signal is read by a CAN node. The unit is 0.1 %.
// The zero value is replaced by 875 (87.5 %) during calculations.
type SamplePoint int

func calcSamplePoint(tseg1, nq int) SamplePoint {
	return SamplePoint(((1+tseg1)*1000 + nq/2) / nq)
}

func (sp *SamplePoint) setupLazy(bitrate uint32) {
	if *sp == 0 {
		*sp = 875
	}
}

func (sp SamplePoint) phSeg2(nq int) int {
	return ((1000-int(sp))*nq + 500) / 1000
}

type CalcOption func(*calcConf)

type calcConf struct {
	preferLowerPrescaler bool
	alignPhSeg1PhSeg2    bool
}

func PreferLowerPrescaler() CalcOption {
	return func(conf *calcConf) {
		conf.preferLowerPrescaler = true
	}
}

func AlignPhSeg1PhSeg2() CalcOption {
	return func(conf *calcConf) {
		conf.alignPhSeg1PhSeg2 = true
	}
}

// CalcBitTiming calculates a bit timing with the desired location of the
// sample point for the given oscillator frequency and bitrate.
// Zero values may be used for the sample point and
// for the resynchronization jump width (sjw),
// in which case sp=87.5 % and sjw=1 are substituted.
func CalcBitTiming(fOsc, bitrate uint32, sp SamplePoint, dev *DevSpec, opts ...CalcOption) (t *BitTiming, err error) {
	var conf calcConf

	for _, o := range opts {
		o(&conf)
	}
	var minErrLoc = 1 << 16
	var bestTiming BitTiming
	nqMax := dev.NqMax()

	incr := minVal(dev.PrescalerIncr)

	nq0 := fOsc / bitrate
	if dev.FOscDiv != 0 {
		nq0 /= uint32(dev.FOscDiv)
	}

	sp.setupLazy(bitrate)

	for preSc := minVal(dev.PrescalerMin); preSc < dev.PrescalerMax; preSc += incr {
		nq := int(nq0 / uint32(preSc))
		if nq0%uint32(preSc) != 0 {
			continue
		}
		if nq <= 0 {
			break
		}
		if nq > nqMax {
			continue
		}
		ps2 := dev.constrainPhSeg2(sp.phSeg2(nq))
	again:
		// compute a tseg1
		tseg1 := nq - ps2 - syncSeg
		if tseg1 > dev.TSeg1Max {
			if ps2 >= dev.TSeg2Max {
				continue
			}
			ps2++
			goto again
		}
		// at least 1tq for each of propSeg and phSeg1 needed
		if tseg1 < 2 {
			break
		}

		errLoc := abs(int(calcSamplePoint(tseg1, nq) - sp))
		if errLoc > minErrLoc {
			continue
		}
		if conf.preferLowerPrescaler {
			if errLoc == minErrLoc {
				continue
			}
		}
		minErrLoc = errLoc

		ps1 := (tseg1 + 1) / 2
		if conf.alignPhSeg1PhSeg2 {
			ps1 = ps2 - 1
		}
		prop, ps1 := dev.SplitTSeg1(tseg1, ps1)

		bestTiming = BitTiming{
			Prescaler: preSc,
			PropSeg:   prop,
			PhaseSeg1: ps1,
			PhaseSeg2: ps2,
		}
	}

	if bestTiming.PhaseSeg2 == 0 {
		return nil, ErrNoValidBitTimingFound
	}

	return &bestTiming, nil
}

var ErrNoValidBitTimingFound = errors.New("can: no valid bit timing found")

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minVal(v int) int {
	return max(1, v)
}

func (dev *DevSpec) SplitTSeg1(tseg1, ps1start int) (prop, ps1 int) {
	ps1 = ps1start

	// propSeg gets the remaining tqs
	prop = tseg1 - ps1

	// adjust prop, if it is negative or zero
	for prop <= 0 {
		ps1--
		prop++
	}

	// adjust prop, if it is too large to be programmable
	maxProp := dev.PropSegMax
	if maxProp == 0 {
		maxProp = dev.TSeg1Max / 2
	}
	for prop > maxProp {
		prop--
		ps1++
	}
	return
}

func (dev *DevSpec) constrainPhSeg2(ps2 int) int {
	minPhSeg2 := minVal(dev.TSeg2Min)

	if ps2 < minPhSeg2 {
		ps2 = minPhSeg2
	}
	if ps2 > dev.TSeg2Max {
		ps2 = dev.TSeg2Max
	}
	return ps2
}

func oscTol(sjw, prop, ps1, ps2 int) int {
	minPh := ps1
	if ps2 < ps1 {
		minPh = ps2
	}
	nq := 1 + prop + ps1 + ps2
	cond1 := 1e6 * minPh / 2 / (13*nq - ps2)
	cond2 := 1e6 * sjw / (20 * nq)
	if cond1 < cond2 {
		return cond1
	}
	return cond2
}
