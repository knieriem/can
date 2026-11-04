package cantest

import (
	"io"
	"sync"
	"testing"

	"github.com/knieriem/can"
)

// bus is an in-memory bus: each message written by one device
// is delivered to all other devices.
type bus struct {
	mu   sync.Mutex
	devs map[*memDev]bool
}

type memDev struct {
	bus  *bus
	c    chan can.Msg
	done chan struct{}
	once sync.Once
	can.Unversioned
}

func (b *bus) open() (can.Device, error) {
	d := &memDev{bus: b, c: make(chan can.Msg, 1024), done: make(chan struct{})}
	b.mu.Lock()
	b.devs[d] = true
	b.mu.Unlock()
	return d, nil
}

func (d *memDev) Read(buf []can.Msg) (int, error) {
	select {
	case m := <-d.c:
		buf[0] = m
		return 1, nil
	case <-d.done:
		return 0, io.EOF
	}
}

func (d *memDev) WriteMsg(m *can.Msg) error {
	d.bus.mu.Lock()
	defer d.bus.mu.Unlock()
	for o := range d.bus.devs {
		if o == d {
			continue
		}
		o.c <- m.Clone()
	}
	return nil
}

func (d *memDev) Write(ms []can.Msg) (int, error) {
	for i := range ms {
		if err := d.WriteMsg(&ms[i]); err != nil {
			return i, err
		}
	}
	return len(ms), nil
}

func (d *memDev) ID() string { return "mem" }

func (d *memDev) Close() error {
	d.once.Do(func() {
		d.bus.mu.Lock()
		delete(d.bus.devs, d)
		d.bus.mu.Unlock()
		close(d.done)
	})
	return nil
}

func TestSuite(t *testing.T) {
	b := &bus{devs: make(map[*memDev]bool)}
	Run(t, b.open, b.open, Options{FD: true})
}
