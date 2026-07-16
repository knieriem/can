package timing

// Frac1000 represents a fraction of 1, scaled to 1000.
// A value of 875 means .875, or 87.5%. The unit is 0.1 % = 1 ‰.
type Frac1000 int

func (f Frac1000) Percent() float64 {
	return float64(f) / 10
}

func calcFrac(part, whole int) Frac1000 {
	return Frac1000((part*1000 + whole/2) / whole)
}

// apply multiplies a value by frac.
func (f Frac1000) apply(v int) int {
	return (int(f)*v + 500) / 1000
}

// applyCompl multiplies a value by (1 - frac).
func (f Frac1000) applyCompl(v int) int {
	return ((1000-int(f))*v + 500) / 1000
}
