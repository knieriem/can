//go:build linux

package socketcan

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

// TestRcvBuf protects setRcvBuf: it must never shrink the default
// buffer, and as root it must reach rcvBufSize. It also logs the size
// the kernel granted, which helps to diagnose dropped frames.
func TestRcvBuf(t *testing.T) {
	fd, err := unix.Socket(unix.AF_CAN, unix.SOCK_RAW, unix.CAN_RAW)
	if err != nil {
		t.Skipf("no CAN socket: %v", err)
	}
	defer unix.Close(fd)

	before, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF)
	if err != nil {
		t.Fatal(err)
	}
	setRcvBuf(fd)
	after, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("rcvbuf: %d bytes, default %d", after, before)
	if after < before {
		t.Errorf("rcvbuf shrank from %d to %d", before, after)
	}
	if os.Geteuid() == 0 && after < rcvBufSize {
		t.Errorf("as root: rcvbuf %d, want at least %d", after, rcvBufSize)
	}
}
