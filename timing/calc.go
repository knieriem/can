package timing

const (
	tq        = 1
	minPhSeg2 = 2 * tq
	maxPhSeg2 = 8 * tq
	maxPhSeg1 = 8 * tq
	maxTSeg1  = maxProp + maxPhSeg1
	maxProp   = 8 * tq
	syncSeg   = 1 * tq
)

// Calculate a CANopen bit timing for a given oscillator frequency
// and the desired bitrate.
func CanOpen(fOsc, bitRate uint32) (*BitTiming, error) {
	return FitSamplePoint(fOsc, bitRate, .875, 1)
}

// Calculate a bit timing with a desired location of the
// sample point for the given oscillator frequency and bitrate.
func FitSamplePoint(fOsc, bitRate uint32, spLoc float32, maxSJW uint) (t *BitTiming, err error) {
	var minErrLoc = 1 << 16
	var bestTiming BitTiming

	for preSc := uint32(1); preSc < 64; preSc++ {
		nq := int(fOsc / bitRate / preSc)
		if fOsc/bitRate%preSc != 0 {
			continue
		}
		if nq <= 0 {
			break
		}
		if nq > 25 {
			continue
		}
		ps2 := calcPhSeg2BySampleLoc(nq, spLoc)
	again:
		// compute a tseg1
		tseg1 := nq - ps2 - syncSeg
		if tseg1 > maxTSeg1 {
			if ps2 >= maxPhSeg2 {
				continue
			}
			ps2++
			goto again
		}
		// we need at least 1tq for each of propSeg and phSeg1
		if tseg1 < 2 {
			break
		}

		// split tseg1, start with phSeg1 one less than phSeg2
		prop, ps1 := splitTSeg1(tseg1, ps2-1)
		if ps1 > maxPhSeg1 { // should never be true
			continue
		}

		errLoc := abs((1+int(tseg1))*1000/int(nq) - int(spLoc*1000))
		if errLoc > minErrLoc {
			continue
		}
		minErrLoc = errLoc

		bestTiming = BitTiming{
			Prescaler: uint(preSc),
			PhaseSeg1: uint(ps1),
			PhaseSeg2: uint(ps2),
			PropSeg:   uint(prop),
		}
	}
	if bestTiming.PhaseSeg2 == 0 {
		err = Error("unable to calculate a bit timing")
	} else {
		t = &bestTiming
		sjw := t.PhaseSeg1
		if sjw > 4 {
			sjw = 4
		}
		if sjw > maxSJW {
			sjw = maxSJW
		}
		t.SJW = sjw
		t.Clock = fOsc
	}
	return
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func splitTSeg1(tseg1, ps1start int) (prop, ps1 int) {
	ps1 = ps1start

	// propSeg gets the remaining tqs
	prop = tseg1 - ps1

	// adjust prop, if it is negative or zero
	for prop <= 0 {
		ps1--
		prop++
	}

	// adjust prop, if it is too large to be programmable
	for prop > maxProp {
		prop--
		ps1++
	}
	return
}

func calcPhSeg2BySampleLoc(nq int, spLoc float32) (ps2 int) {
	ps2 = int((1-spLoc)*float32(nq) + .5)
	if ps2 < minPhSeg2 {
		ps2 = minPhSeg2
	}
	if ps2 > maxPhSeg2 {
		ps2 = maxPhSeg2
	}
	return
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
