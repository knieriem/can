package timing

// Extract a bit timing from a SJA1000 BTR0BTR1 register value
func SJA1000(fOsc uint32, btr0btr1 uint) (t *BitTiming) {
	t = new(BitTiming)

	t.Clock = fOsc

	btr0 := btr0btr1 >> 8
	t.SJW = ((btr0 >> 6) & 3) + 1
	t.Prescaler = (btr0 & 0x3F) + 1

	btr1 := btr0btr1 & 0xFF
	t.PhaseSeg2 = ((btr1 >> 4) & 7) + 1
	tseg1 := int((btr1 & 0xF) + 1)

	prop, ps1 := splitTSeg1(tseg1, int(t.PhaseSeg2)-1)
	t.TripleSampling = btr1&(1<<7) != 0
	t.PhaseSeg1 = uint(ps1)
	t.PropSeg = uint(prop)
	return
}
