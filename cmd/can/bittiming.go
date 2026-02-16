package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/pcan"
	"github.com/knieriem/can/timing"
	"github.com/knieriem/can/timing/dev"
	"github.com/knieriem/tool"
)

var cmdBittiming = &tool.Command{
	UsageLine:    "bt config",
	Short:        "resolve bit timing configuration",
	Long:         ``,
	ExtraArgsReq: 1,
}

func init() {
	cmdBittiming.Flag.Float64Var(&clockFlag, "clock", 0, "oscillator frequency (MHz)")
	cmdBittiming.Flag.StringVar(&devSpec, "dev", "sja1000", "device")

	cmdBittiming.Run = runBittiming
}

var devSpecMap = map[string]*timing.Controller{
	"mcp2515":   dev.MCP2515,
	"mcp2518fd": dev.MCP2518FD,
	"pcanfd":    &pcan.DevSpecFD,
	"sja1000":   dev.SJA1000,

	"candlelightfd": dev.CandleLightFD,
}

var clockFlag float64
var devSpec string

func runBittiming(cmd *tool.Command, w io.Writer, args []string) (err error) {
	c, err := can.ParseConfig(args...)
	if err != nil {
		return err
	}

	dev, ok := devSpecMap[devSpec]
	if !ok {
		return errors.New("device unknown")
	}
	clock := dev.Clock
	if clockFlag != 0. {
		clock = uint32(clockFlag)
	}
	err = doBittiming("nominal", clock, &c.Nominal, &dev.Nominal)
	if err != nil {
		return err
	}

	if c.Data.Valid {
		if dev.Data == nil {
			return errors.New("device not fd capable")
		}
		err = doBittiming("data", clock, &c.Data.Value, dev.Data)
		if err != nil {
			return err
		}
	}

	fmt.Println(c.Format(" "))

	c.Nominal.Bitrate = 0
	c.Data.Value.Bitrate = 0
	fmt.Println(c.Format(" "))

	return nil
}

func doBittiming(kind string, clock uint32, c *can.BitTimingConfig, constr *timing.Constraints) error {
	err := c.Resolve(nil, clock, constr)
	if err != nil {
		return err
	}
	c.Bitrate = c.BitTiming.Bitrate(clock)

	fmt.Println(kind)
	nq := c.Nq()

	fmt.Printf("\tbitrate: %v bps\n", can.FormatBitrate(c.BitTiming.Bitrate(clock)))
	fmt.Printf("\tbit: %d tq (%v)\n", nq, time.Duration(nq)*c.Tq)
	fmt.Printf("\ttq: %v\n", c.Tq)
	fmt.Printf("\tbrp: %d\n", c.Prescaler)
	fmt.Printf("\ttseg1: %d tq (%d tq + %d tq)\n", c.PropSeg+c.PhaseSeg1, c.PropSeg, c.PhaseSeg1)
	fmt.Printf("\ttseg2: %d tq\n", c.PhaseSeg2)
	fmt.Printf("\tsample-point: %v %%\n", c.BitTiming.SamplePoint().Percent())
	fmt.Printf("\tsjw: %d tq (%.1f%%)\n", c.SJW, float64(c.SJW*100)/float64(nq))
	return nil
}
