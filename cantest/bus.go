package cantest

import (
	"sync"

	"github.com/knieriem/can"
)

// A Bus is an in-memory CAN bus for tests that need no hardware.
// A message written to one of its devices is received by all others.
// Like a real controller, a device whose receive queue is full
// drops further messages; the writer does not notice.
type Bus struct {
	mu   sync.Mutex
	devs map[*busDev]bool
}

// NewBus returns an empty bus.
func NewBus() *Bus {
	return &Bus{devs: make(map[*busDev]bool)}
}

// Open attaches a new device to b.
func (b *Bus) Open() (can.Device, error) {
	d := &busDev{bus: b, c: make(chan can.Msg, 1024), done: make(chan struct{})}
	b.mu.Lock()
	b.devs[d] = true
	b.mu.Unlock()
	return d, nil
}

type busDev struct {
	bus  *Bus
	c    chan can.Msg
	done chan struct{}
	once sync.Once
	can.Unversioned
}

func (d *busDev) closed() bool {
	select {
	case <-d.done:
		return true
	default:
		return false
	}
}

func (d *busDev) Read(buf []can.Msg) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	select {
	case m := <-d.c:
		buf[0] = m
	case <-d.done:
		return 0, can.ErrClosed
	}
	n := 1
	for n < len(buf) {
		select {
		case buf[n] = <-d.c:
			n++
		default:
			return n, nil
		}
	}
	return n, nil
}

func (d *busDev) WriteMsg(m *can.Msg) error {
	if d.closed() {
		return can.ErrClosed
	}
	d.bus.mu.Lock()
	defer d.bus.mu.Unlock()
	for o := range d.bus.devs {
		if o == d {
			continue
		}
		select {
		case o.c <- m.Clone():
		default: // receive queue full: drop
		}
	}
	return nil
}

func (d *busDev) Write(msgs []can.Msg) (int, error) {
	for i := range msgs {
		if err := d.WriteMsg(&msgs[i]); err != nil {
			return i, err
		}
	}
	return len(msgs), nil
}

func (d *busDev) ID() string { return "bus" }

func (d *busDev) Close() error {
	var err error = can.ErrClosed
	d.once.Do(func() {
		d.bus.mu.Lock()
		delete(d.bus.devs, d)
		d.bus.mu.Unlock()
		close(d.done)
		err = nil
	})
	return err
}
