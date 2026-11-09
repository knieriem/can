//go:build linux

package socketcan

import (
	"testing"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/socketcan/internal/linux"
)

// TestDecodeStatus protects the translation of error frames, which
// the end-to-end tests cannot reach: vcan produces no error frames.
// Each case starts from the state the previous one left.
func TestDecodeStatus(t *testing.T) {
	type data struct{ ctrl, protLoc, txErr, rxErr byte }
	var state can.BusState
	for _, c := range []struct {
		name  string
		class uint32
		d     data
		want  can.Status
	}{
		{"ack", linux.CAN_ERR_ACK, data{}, can.Status{Events: can.NoAck}},
		{"prot in ack slot", linux.CAN_ERR_PROT, data{protLoc: errProtLocAck}, can.Status{Events: can.NoAck}},
		{"other prot", linux.CAN_ERR_PROT, data{protLoc: 0x08}, can.Status{Events: can.BusError}},
		{"rx warning", linux.CAN_ERR_CRTL, data{ctrl: linux.CAN_ERR_CRTL_RX_WARNING}, can.Status{State: can.ErrorWarning}},
		{"state is kept", linux.CAN_ERR_ACK, data{}, can.Status{State: can.ErrorWarning, Events: can.NoAck}},
		{"tx passive, counters", linux.CAN_ERR_CRTL | errCnt, data{ctrl: linux.CAN_ERR_CRTL_TX_PASSIVE, txErr: 130, rxErr: 2},
			can.Status{State: can.ErrorPassive, TxErr: 130, RxErr: 2, Counters: true}},
		{"rx overflow", linux.CAN_ERR_CRTL, data{ctrl: linux.CAN_ERR_CRTL_RX_OVERFLOW}, can.Status{State: can.ErrorPassive, Events: can.RxOverflow}},
		{"bus-off", linux.CAN_ERR_BUSOFF, data{}, can.Status{State: can.BusOff}},
		{"restarted", linux.CAN_ERR_RESTARTED, data{}, can.Status{Events: can.Restarted}},
		{"back to active", linux.CAN_ERR_CRTL, data{ctrl: errCrtlActive}, can.Status{}},
	} {
		var f frame
		f.setid(linux.CAN_ERR_FLAG | c.class)
		f.setLen(8)
		d := f.data()
		d[1], d[3], d[6], d[7] = c.d.ctrl, c.d.protLoc, c.d.txErr, c.d.rxErr
		if got := f.decodeStatus(&state); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
