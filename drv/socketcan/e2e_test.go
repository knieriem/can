//go:build linux

package socketcan_test

import (
	"os"
	"strings"
	"testing"

	"github.com/knieriem/can"
	"github.com/knieriem/can/cantest"
	"github.com/knieriem/can/drv/socketcan"
)

func init() {
	can.RegisterDriver(socketcan.Driver)
}

// TestVCAN runs the end-to-end suite on a vcan interface,
// vcan0 unless CAN_TEST_VCAN names another one. Setup:
//
//	ip link add dev vcan0 type vcan
//	ip link set vcan0 mtu 72	# enables the FD cases
//	ip link set up vcan0
func TestVCAN(t *testing.T) {
	name := os.Getenv("CAN_TEST_VCAN")
	if name == "" {
		name = "vcan0"
	}
	mtu, err := os.ReadFile("/sys/class/net/" + name + "/mtu")
	if err != nil {
		t.Skipf("no interface %s", name)
	}
	open := func() (can.Device, error) {
		return can.Open("socketcan:" + name)
	}
	fd := strings.TrimSpace(string(mtu)) == "72"
	cantest.Run(t, open, open, cantest.Options{FD: fd})
}

// TestPair runs the suite on two real adapters wired to the same bus.
// CAN_TEST_PAIR holds two device specs separated by "|", for example
//
//	CAN_TEST_PAIR='socketcan:can0,500k,db:2M|socketcan:can1,500k,db:2M'
//
// Set CAN_TEST_FD=1 if both adapters run in FD mode.
func TestPair(t *testing.T) {
	pair := os.Getenv("CAN_TEST_PAIR")
	if pair == "" {
		t.Skip("CAN_TEST_PAIR not set")
	}
	tx, rx, ok := strings.Cut(pair, "|")
	if !ok {
		t.Fatalf("CAN_TEST_PAIR: want two specs separated by |, got %q", pair)
	}
	opener := func(spec string) cantest.Opener {
		return func() (can.Device, error) { return can.Open(spec) }
	}
	cantest.Run(t, opener(tx), opener(rx), cantest.Options{FD: os.Getenv("CAN_TEST_FD") == "1"})
}
