package dev

import "github.com/knieriem/can/timing"

// CandleLightFD defines the register value constraints of the [candleLight FD]
// USB adapter, which is based on the STM32G0B1 controller
// that contains an [M_CAN] peripheral.
//
// [candleLight FD]: https://linux-automation.com/en/products/candlelight-fd.html
// [M_CAN]: https://www.bosch-semiconductors.com/products/ip-modules/can-ip-modules/m-can/
var CandleLightFD = &timing.Controller{
	Clock: 40e6,
	Nominal: timing.Constraints{
		TSeg1Max:     256,
		TSeg2Max:     128,
		SJWMax:       128,
		PrescalerMax: 512,
	},
	Data: &timing.Constraints{
		TSeg1Max:     32,
		TSeg2Max:     16,
		SJWMax:       16,
		PrescalerMax: 32,
	},
}
