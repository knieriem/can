package canrpc

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"testing"
	"time"

	"github.com/knieriem/can"
	"github.com/knieriem/can/cantest"
)

// fakeDev delivers messages sent on c; Read fails with the
// first error sent on errc, and with every later call.
type fakeDev struct {
	c    chan can.Msg
	errc chan error
	err  error
	can.Unversioned
}

func newFakeDev() *fakeDev {
	return &fakeDev{c: make(chan can.Msg), errc: make(chan error)}
}

func (d *fakeDev) Read(buf []can.Msg) (int, error) {
	if d.err != nil {
		return 0, d.err
	}
	select {
	case m := <-d.c:
		buf[0] = m.Clone()
		return 1, nil
	case d.err = <-d.errc:
		return 0, d.err
	}
}

func (d *fakeDev) WriteMsg(*can.Msg) error { return nil }
func (d *fakeDev) ID() string              { return "fake" }
func (d *fakeDev) Close() error            { return nil }

// connect registers dev with an RPC server and returns a client device.
func connect(t *testing.T, dev can.Device) can.Device {
	t.Helper()
	srv := rpc.NewServer()
	if err := Register(srv, dev, ""); err != nil {
		t.Fatal(err)
	}
	c1, c2 := net.Pipe()
	go srv.ServeConn(c1)
	cl := rpc.NewClient(c2)
	t.Cleanup(func() { cl.Close() })
	return NewDevice(cl, "")
}

func readTimeout(d can.Device) (can.Msg, error) {
	type result struct {
		m   can.Msg
		err error
	}
	c := make(chan result, 1)
	go func() {
		buf := make([]can.Msg, 4)
		n, err := d.Read(buf)
		var r result
		if n > 0 {
			r.m = buf[0].Clone()
		}
		r.err = err
		c <- r
	}()
	select {
	case r := <-c:
		return r.m, r.err
	case <-time.After(2 * time.Second):
		return can.Msg{}, errors.New("timeout: Read blocked")
	}
}

// TestReadErrorNoHang protects the fix for a server hang: after the
// server-side device fails, every Read returns the error. Before, the
// second Read blocked forever. The suite never makes that device fail.
func TestReadErrorNoHang(t *testing.T) {
	fd := newFakeDev()
	d := connect(t, fd)

	var m can.Msg
	m.Id = 0x123
	m.SetData(make([]byte, 12))
	go func() { fd.c <- m }()
	got, err := readTimeout(d)
	if err != nil || got.Id != 0x123 || len(got.Data()) != 12 {
		t.Fatalf("got id=%#x len=%d err=%v", got.Id, len(got.Data()), err)
	}

	go func() { fd.errc <- errors.New("unplugged") }()
	for i := range 3 {
		if _, err := readTimeout(d); err == nil || err.Error() == "timeout: Read blocked" {
			t.Fatalf("read %d after device error: %v", i, err)
		}
	}
}

// TestServerErrClosed protects the error mapping across RPC: a closed
// server-side device is reported as ErrClosed, although net/rpc
// transports only the error text.
func TestServerErrClosed(t *testing.T) {
	fd := newFakeDev()
	d := connect(t, fd)
	go func() { fd.errc <- fmt.Errorf("socketcan: read: %w", can.ErrClosed) }()
	if _, err := readTimeout(d); !errors.Is(err, can.ErrClosed) {
		t.Errorf("got %v, want ErrClosed", err)
	}
}

// TestSuite runs the cantest suite through RPC connections to devices
// on an in-memory bus. It covers round trips, concurrency and Close.
func TestSuite(t *testing.T) {
	bus := cantest.NewBus()
	open := func() (can.Device, error) {
		dev, err := bus.Open()
		if err != nil {
			return nil, err
		}
		t.Cleanup(func() { dev.Close() })
		return connect(t, dev), nil
	}
	cantest.Run(t, open, open, cantest.Options{FD: true})
}
