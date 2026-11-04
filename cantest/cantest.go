// Package cantest provides an end-to-end test suite for [can.Device]
// implementations.
//
// The suite needs two devices attached to the same bus: a virtual one
// such as a SocketCAN vcan interface, or two real adapters wired
// together. Messages written to one device must be received by the
// other. A driver's tests call [Run] with functions that open
// the two devices; each subtest opens and closes its own pair.
package cantest

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/knieriem/can"
)

// An Opener opens a device. It is called once per subtest.
type Opener func() (can.Device, error)

// Options adjust the suite to the devices under test.
type Options struct {
	// FD enables the CAN FD cases.
	FD bool

	// Timeout limits how long the suite waits for a message.
	// The default is one second.
	Timeout time.Duration
}

// Run runs the suite. Messages are written to devices opened
// by openTX and expected on devices opened by openRX.
func Run(t *testing.T, openTX, openRX Opener, opt Options) {
	if opt.Timeout == 0 {
		opt.Timeout = time.Second
	}
	s := &suite{openTX: openTX, openRX: openRX, opt: opt}
	t.Run("RoundTrip", s.testRoundTrip)
	t.Run("ConcurrentWrite", s.testConcurrentWrite)
	t.Run("CloseUnblocksRead", s.testCloseUnblocksRead)
}

type suite struct {
	openTX, openRX Opener
	opt            Options
}

func (s *suite) open(t *testing.T, o Opener) can.Device {
	t.Helper()
	d, err := o()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

// Case is a message the suite sends, written as a [can.Msg.FromExpr]
// expression, for example "123#AABB", "1234567#R", or "123##1" followed
// by FD payload.
type Case struct {
	Expr string
	FD   bool
}

// Cases returns the round trip cases. Each standard and extended
// identifier ends in every hex digit once, because a wrong
// flag decoding typically depends on single identifier bits.
func Cases() []Case {
	var cs []Case
	for i := range 16 {
		cs = append(cs,
			Case{Expr: fmt.Sprintf("45%X#AABB", i)},
			Case{Expr: fmt.Sprintf("18DAF10%X#11223344", i)},
		)
	}
	cs = append(cs,
		Case{Expr: "7FF#"},
		Case{Expr: "000#0102030405060708"},
		Case{Expr: "1FFFFFFF#01"},
		Case{Expr: "00000123#01"}, // extended frame with a small ID
		Case{Expr: "123#R"},
		Case{Expr: "18DAF105#R"},

		Case{Expr: "456##0AABB", FD: true},
		Case{Expr: "456##1AABB", FD: true},
		Case{Expr: "18DAF105##1AABB", FD: true},
		Case{Expr: "123##1" + payload(12), FD: true},
		Case{Expr: "18DAF105##0" + payload(64), FD: true},
		Case{Expr: "18DAF105##1" + payload(64), FD: true},
	)
	return cs
}

func payload(n int) string {
	var b bytes.Buffer
	for i := range n {
		fmt.Fprintf(&b, "%02X", i)
	}
	return b.String()
}

func (s *suite) testRoundTrip(t *testing.T) {
	tx := s.open(t, s.openTX)
	rx := s.open(t, s.openRX)
	r := newReader(rx)

	for _, c := range Cases() {
		if c.FD && !s.opt.FD {
			continue
		}
		var want can.Msg
		if err := want.FromExpr(c.Expr); err != nil {
			t.Fatalf("%s: %v", c.Expr, err)
		}
		if err := tx.WriteMsg(&want); err != nil {
			t.Errorf("%s: write: %v", c.Expr, err)
			continue
		}
		got, err := r.next(s.opt.Timeout)
		if err != nil {
			t.Fatalf("%s: read: %v", c.Expr, err)
		}
		if !equal(&got, &want) {
			t.Errorf("%s: got id=%#x flags=%#x data=% x, want id=%#x flags=%#x data=% x",
				c.Expr, got.Id, flags(&got), got.Data(), want.Id, flags(&want), want.Data())
		}
	}
}

// testConcurrentWrite writes from several goroutines to one device.
// Run with -race. Messages of one writer must arrive complete and in order.
func (s *suite) testConcurrentWrite(t *testing.T) {
	const (
		writers = 4
		perW    = 200
	)
	tx := s.open(t, s.openTX)
	rx := s.open(t, s.openRX)
	r := newReader(rx)

	var wg sync.WaitGroup
	errc := make(chan error, writers)
	for w := range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var m can.Msg
			m.Id = 0x100 + uint32(w)
			var data [2]byte
			for i := range perW {
				binary.BigEndian.PutUint16(data[:], uint16(i))
				m.SetData(data[:])
				if err := writeRetry(tx, &m); err != nil {
					errc <- fmt.Errorf("writer %d, msg %d: %w", w, i, err)
					return
				}
			}
		}()
	}

	next := make(map[uint32]int)
	for range writers * perW {
		m, err := r.next(s.opt.Timeout)
		if err != nil {
			t.Fatalf("read after %v messages: %v", next, err)
		}
		seq := int(binary.BigEndian.Uint16(m.Data()))
		if seq != next[m.Id] {
			t.Fatalf("id %#x: got seq %d, want %d", m.Id, seq, next[m.Id])
		}
		next[m.Id]++
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		t.Error(err)
	}
}

// writeRetry retries while the transmit queue is full,
// which real adapters report under load.
func writeRetry(d can.Device, m *can.Msg) error {
	for range 100 {
		err := d.WriteMsg(m)
		if !errors.Is(err, can.ErrTxQueueFull) {
			return err
		}
		time.Sleep(time.Millisecond)
	}
	return can.ErrTxQueueFull
}

func (s *suite) testCloseUnblocksRead(t *testing.T) {
	d, err := s.openRX()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		buf := make([]can.Msg, 1)
		for {
			if _, err := d.Read(buf); err != nil {
				done <- err
				return
			}
		}
	}()
	time.Sleep(50 * time.Millisecond)
	if err := d.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
	select {
	case err := <-done:
		t.Logf("read returned: %v", err)
	case <-time.After(s.opt.Timeout):
		t.Fatal("read still blocked after close")
	}
}

// reader reads data messages from a device in its own goroutine,
// so that tests can wait with a timeout. Status messages are skipped.
type reader struct {
	c chan result
}

type result struct {
	m   can.Msg
	err error
}

func newReader(d can.Device) *reader {
	r := &reader{c: make(chan result, 64)}
	go func() {
		buf := make([]can.Msg, 8)
		for {
			n, err := d.Read(buf)
			for i := range buf[:n] {
				if buf[i].IsStatus() {
					continue
				}
				r.c <- result{m: buf[i].Clone()}
			}
			if err != nil {
				r.c <- result{err: err}
				close(r.c)
				return
			}
		}
	}()
	return r
}

func (r *reader) next(timeout time.Duration) (can.Msg, error) {
	select {
	case res, ok := <-r.c:
		if !ok {
			return can.Msg{}, errors.New("device closed")
		}
		return res.m, res.err
	case <-time.After(timeout):
		return can.Msg{}, errors.New("timeout")
	}
}

// flags returns the flags relevant for comparing data messages.
// ForceFD is implied by a payload longer than eight bytes.
func flags(m *can.Msg) can.Flags {
	f := m.Flags & (can.ExtFrame | can.RTRMsg | can.FDSwitchBitrate | can.ForceFD)
	if len(m.Data()) > 8 {
		f &^= can.ForceFD
	}
	return f
}

func equal(a, b *can.Msg) bool {
	return a.Id == b.Id && flags(a) == flags(b) && bytes.Equal(a.Data(), b.Data())
}
