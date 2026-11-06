package cantest

import "testing"

func TestBus(t *testing.T) {
	b := NewBus()
	Run(t, b.Open, b.Open, Options{FD: true})
}
