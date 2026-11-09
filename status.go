package can

import "time"

// statusLevel selects the status messages that Read delivers.
type statusLevel int

const (
	noStatusMsgs statusLevel = iota
	statusMsgs
	allStatusMsgs
)

// WithStatusMsgs makes Read deliver status messages: when the bus
// state changes, when a transmitted frame is not acknowledged (NoAck,
// at most once per second, as the controller retries endlessly), when
// received frames were lost, and when the controller restarts.
// Without this option or WithBusErrors, Read delivers no status messages.
func WithStatusMsgs() Option {
	return func(p *openProps) {
		p.status = statusMsgs
	}
}

// WithBusErrors makes Read deliver all status messages the driver
// reports, including one for every bus error. It is meant for
// diagnosis, and for servers that pass status messages on, such as
// those of package canrpc.
func WithBusErrors() Option {
	return func(p *openProps) {
		p.status = allStatusMsgs
	}
}

// withStatusFilter returns d, wrapped in a filter for level unless
// the program wants all status messages.
func withStatusFilter(d Device, level statusLevel) Device {
	if level == allStatusMsgs {
		return d
	}
	return &statusFilter{Device: d, level: level, now: time.Now}
}

// statusFilter drops the status messages a program did not ask for.
// Drivers report all of them; filtering here gives all drivers the
// same behavior.
type statusFilter struct {
	Device
	level     statusLevel
	state     BusState // last delivered state
	lastNoAck time.Time
	now       func() time.Time
}

func (f *statusFilter) Read(buf []Msg) (int, error) {
	for {
		n, err := f.Device.Read(buf)
		n = f.filter(buf[:n])
		if n > 0 || err != nil {
			return n, err
		}
	}
}

// filter drops unwanted status messages from msgs and returns the
// number of messages kept, which keep their order. It swaps rather
// than copies, so that no two messages share payload storage.
func (f *statusFilter) filter(msgs []Msg) int {
	n := 0
	for i := range msgs {
		if !f.keep(&msgs[i]) {
			continue
		}
		if n != i {
			msgs[n], msgs[i] = msgs[i], msgs[n]
		}
		n++
	}
	return n
}

// keep reports whether m is to be delivered. It removes the events
// of a status message that are not to be reported.
func (f *statusFilter) keep(m *Msg) bool {
	st, ok := m.Status()
	if !ok {
		return true
	}
	if f.level == noStatusMsgs {
		return false
	}
	st.Events &^= BusError
	if st.Events&NoAck != 0 {
		if now := f.now(); now.Sub(f.lastNoAck) < time.Second {
			st.Events &^= NoAck
		} else {
			f.lastNoAck = now
		}
	}
	if st.Events == 0 && st.State == f.state {
		return false
	}
	f.state = st.State
	m.status = st
	return true
}
