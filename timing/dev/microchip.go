package dev

import "github.com/knieriem/can/timing"

// MCP2515Const defines the register value constraints of the MCP2515 CAN
// controller;
// see https://ww1.microchip.com/downloads/en/DeviceDoc/MCP2515-Stand-Alone-CAN-Controller-with-SPI-20001801J.pdf#page=43
var MCP2515 = &timing.DevSpec{
	PropSegMax:   8,
	TSeg1Max:     16,
	TSeg2Min:     2,
	TSeg2Max:     8,
	SJWMax:       4,
	PrescalerMax: 64,

	FOscDiv: 2,

	EncodeToReg: func(bt *timing.BitTiming) *timing.RegValue {
		cnf1 := byte(bt.SJW-1)<<6 | byte(bt.Prescaler-1)
		cnf2 := 1<<7 | byte(bt.PhaseSeg1-1)<<3 | byte(bt.PropSeg-1)
		cnf3 := byte(bt.PhaseSeg2 - 1)
		return &timing.RegValue{
			Reg8: []byte{cnf1, cnf2, cnf3},
		}
	},
}

// MCP2518FD defines the register value constraints of the MCP2518FD CAN
// controller;
// see https://ww1.microchip.com/downloads/aemDocuments/documents/OTH/ProductDocuments/DataSheets/External-CAN-FD-Controller-with-SPI-Interface-DS20006027B.pdf#page=28
var MCP2518FD = &timing.DevSpecFD{
	Nominal: timing.DevSpec{
		TSeg1Max:     256,
		TSeg2Max:     128,
		SJWMax:       128,
		PrescalerMax: 256,
	},
	Data: timing.DevSpec{
		TSeg1Max:     32,
		TSeg2Max:     16,
		SJWMax:       16,
		PrescalerMax: 256,
	},
}
