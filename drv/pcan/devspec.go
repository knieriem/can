package pcan

import "github.com/knieriem/can/timing"

// DevSpecFD describes the bit timing constraints of the
// PCAN-USB FD's controller. It is defined on all platforms,
// so that tools can compute bit timings for the adapter.
var DevSpecFD = timing.Controller{
	Clock: 80e6,
	Nominal: timing.Constraints{
		TSeg1Max:     256,
		TSeg2Max:     128,
		SJWMax:       128,
		PrescalerMin: 1,
		PrescalerMax: 1024,
	},
	Data: &timing.Constraints{
		TSeg1Max:     32,
		TSeg2Max:     16,
		SJWMax:       16,
		PrescalerMin: 1,
		PrescalerMax: 1024,
	},
}
