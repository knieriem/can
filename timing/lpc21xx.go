package timing

// Extract a bit timing from an LPC21xx BTR register value
func LPC21xx(fOsc uint32, btr uint32) (t *BitTiming) {
	t = new(BitTiming)
	t.Clock = fOsc
	t.Prescaler = uint((btr & 0x1FF) + 1)
	t.SJW = uint(((btr >> 14) & 3) + 1)

	tseg1 := uint(((btr >> 16) & 0xF) + 1)
	t.PropSeg = 1
	t.PhaseSeg1 = tseg1 - 1

	t.PhaseSeg2 = uint(((btr >> 20) & 0x7) + 1)
	return
}

func (t *BitTiming) LPC21xxBTR() (r uint32) {
	r = uint32((t.PropSeg+t.PhaseSeg1-1)&0xF) << 16
	r |= uint32((t.PhaseSeg2-1)&7) << 20
	r |= uint32(t.Prescaler - 1)
	r |= uint32((t.SJW-1)&3) << 14
	return
}
