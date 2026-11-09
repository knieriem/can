//go:build linux

package socketcan

import (
	"encoding/binary"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

// TestRxqDrops protects the parsing of the kernel's drop counter
// (SO_RXQ_OVFL), which only an actual overflow would exercise.
func TestRxqDrops(t *testing.T) {
	oob := make([]byte, 2*unix.CmsgSpace(4))
	put := func(b []byte, typ int32, v uint32) {
		h := (*unix.Cmsghdr)(unsafe.Pointer(&b[0]))
		h.Level = unix.SOL_SOCKET
		h.Type = typ
		h.SetLen(unix.CmsgLen(4))
		binary.NativeEndian.PutUint32(b[unix.CmsgLen(0):], v)
	}
	put(oob, unix.SO_TIMESTAMP, 1) // another message first
	put(oob[unix.CmsgSpace(4):], unix.SO_RXQ_OVFL, 42)

	if drops, ok := rxqDrops(oob); !ok || drops != 42 {
		t.Errorf("got %d, %v; want 42, true", drops, ok)
	}
	if _, ok := rxqDrops(oob[:unix.CmsgSpace(4)]); ok {
		t.Error("found a counter in a buffer without one")
	}
}
