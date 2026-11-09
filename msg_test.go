package can

import (
	"bytes"
	"testing"
)

// TestSetData protects the basic round trip of SetData and Data
// across inline and external storage.
func TestSetData(t *testing.T) {
	long := bytes.Repeat([]byte{0xAA}, 64)
	var m Msg
	for _, b := range [][]byte{nil, {1}, {1, 2, 3, 4, 5, 6, 7, 8}, long[:12], {9, 9}, long, {}} {
		m.SetData(b)
		if !bytes.Equal(m.Data(), b) {
			t.Errorf("SetData(% x): Data() = % x", b, m.Data())
		}
	}
}

// TestSetDataAlias protects the in-place idiom: SetData with a slice
// of m's own storage, as returned by Data, only updates the length.
func TestSetDataAlias(t *testing.T) {
	var m Msg
	if d := m.Data(); len(d) != 0 || cap(d) != 8 {
		t.Fatalf("fresh Msg: len %d cap %d, want 0 and 8", len(d), cap(d))
	}

	// classic in place
	d := m.Data()[:3]
	copy(d, []byte{1, 2, 3})
	m.SetData(d)
	if !bytes.Equal(m.Data(), []byte{1, 2, 3}) {
		t.Errorf("classic: Data() = % x", m.Data())
	}

	// FD in place, once ext holds a long payload
	m.SetData(make([]byte, 64))
	d = m.Data()[:12]
	for i := range d {
		d[i] = byte(i)
	}
	m.SetData(d)
	if len(m.Data()) != 12 || m.Data()[11] != 11 {
		t.Errorf("fd: Data() = % x", m.Data())
	}

	// a short slice of ext moves the payload inline
	m.SetData(m.Data()[2:5])
	if !bytes.Equal(m.Data(), []byte{2, 3, 4}) {
		t.Errorf("ext to inline: Data() = % x", m.Data())
	}

	// growing a slice of ext's storage again, within its capacity
	m.SetData(make([]byte, 20))
	d = m.Data()
	m.SetData(d[:cap(d)])
	if len(m.Data()) != cap(d) {
		t.Errorf("regrow: len %d, want %d", len(m.Data()), cap(d))
	}
}

// TestCopySemantics protects the documented copy behavior: a plain copy
// has its own classic payload; Clone also copies long payloads.
func TestCopySemantics(t *testing.T) {
	var a Msg
	a.SetData([]byte{1, 2})
	b := a
	b.SetData([]byte{3, 4})
	if a.Data()[0] != 1 {
		t.Error("copy of a classic Msg shares its payload")
	}

	a.SetData(bytes.Repeat([]byte{1}, 12))
	c := a.Clone()
	c.SetData(bytes.Repeat([]byte{2}, 12))
	if a.Data()[0] != 1 {
		t.Error("Clone shares the payload")
	}
}

// TestReset protects that Reset clears the message but keeps
// the payload storage for reuse.
func TestReset(t *testing.T) {
	var m Msg
	m.Id = 0x123
	m.Flags = ExtFrame
	m.SetData(make([]byte, 64))
	m.Reset()
	if m.Id != 0 || m.Flags != 0 || len(m.Data()) != 0 {
		t.Errorf("after Reset: id=%#x flags=%#x len=%d", m.Id, m.Flags, len(m.Data()))
	}
	if cap(m.ext) < 64 {
		t.Error("Reset dropped the payload storage")
	}
}

// TestAllocs protects the reason for Msg's design: no allocations
// after warm-up, for classic, FD, mixed and in-place use.
func TestAllocs(t *testing.T) {
	var m Msg
	fd := make([]byte, 64)
	m.SetData(fd) // first long payload allocates storage

	for _, tc := range []struct {
		name string
		f    func()
	}{
		{"classic literal", func() { m.SetData([]byte{1, 2, 3}) }},
		{"classic array", func() { var b [8]byte; b[0] = 1; m.SetData(b[:2]) }},
		{"fd reuse", func() { m.SetData(fd) }},
		{"mixed", func() { m.SetData(fd[:2]); m.SetData(fd[:48]) }},
		{"classic in place", func() { m.SetData(fd[:1]); d := m.Data()[:4]; d[3] = 1; m.SetData(d) }},
		{"fd in place", func() { m.SetData(fd); d := m.Data()[:32]; d[31] = 1; m.SetData(d) }},
	} {
		if n := testing.AllocsPerRun(100, tc.f); n != 0 {
			t.Errorf("%s: %v allocs, want 0", tc.name, n)
		}
	}
}
