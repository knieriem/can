package can

import (
	"slices"
	"testing"
	"time"
)

// TestStatusFilter protects what WithStatusMsgs promises: no bus
// errors, NoAck at most once per second, and state messages only on
// changes. Without an option, no status message passes.
func TestStatusFilter(t *testing.T) {
	status := func(s BusState, e Events) Msg {
		var m Msg
		m.SetStatus(Status{State: s, Events: e})
		return m
	}
	var frame Msg
	frame.Id = 0x123
	input := func() []Msg {
		return []Msg{
			status(ErrorActive, NoAck),     // at 0 s
			frame,                          //
			status(ErrorActive, NoAck),     // at 0.5 s: repeated NoAck
			status(ErrorWarning, NoAck),    // at 0.6 s: state change
			status(ErrorWarning, BusError), // no change but a bus error
			status(ErrorWarning, NoAck),    // at 1.2 s
		}
	}
	format := func(msgs []Msg) []string {
		var s []string
		for i := range msgs {
			s = append(s, msgs[i].Format())
		}
		return s
	}

	// The clock advances on each call; the filter asks only for NoAck.
	var clock []time.Duration
	f := &statusFilter{level: statusMsgs, now: func() time.Time {
		d := clock[0]
		clock = clock[1:]
		return time.Unix(1000, 0).Add(d)
	}}
	clock = []time.Duration{0, 500 * time.Millisecond, 600 * time.Millisecond, 1200 * time.Millisecond}
	msgs := input()
	want := []string{
		"status: error-active no-ack",
		"123:",
		"status: error-warning",
		"status: error-warning no-ack",
	}
	if got := format(msgs[:f.filter(msgs)]); !slices.Equal(got, want) {
		t.Errorf("WithStatusMsgs:\ngot  %q\nwant %q", got, want)
	}

	f = &statusFilter{level: noStatusMsgs}
	msgs = input()
	if got := format(msgs[:f.filter(msgs)]); !slices.Equal(got, []string{"123:"}) {
		t.Errorf("no option: got %q", got)
	}
}
