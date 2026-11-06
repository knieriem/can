//go:build linux

package socketcan

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestRcvBuf(t *testing.T) {
	fd, err := unix.Socket(unix.AF_CAN, unix.SOCK_RAW, unix.CAN_RAW)
	if err != nil {
		t.Skipf("no CAN socket: %v", err)
	}
	defer unix.Close(fd)

	setRcvBuf(fd)
	n, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("rcvbuf: %d bytes", n) // the kernel reports twice the requested size
	if os.Geteuid() == 0 && n < rcvBufSize {
		t.Errorf("privileged: rcvbuf %d, want at least %d", n, rcvBufSize)
	}
}
