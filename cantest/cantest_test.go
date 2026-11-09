package cantest

import "testing"

// TestBus runs the suite on the in-memory Bus, which checks both
// the suite itself and the Bus that drivers may test against.
func TestBus(t *testing.T) {
	b := NewBus()
	Run(t, b.Open, b.Open, Options{FD: true})
}
