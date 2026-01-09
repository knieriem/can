package main

import (
	"io"

	"github.com/knieriem/can"
	"github.com/knieriem/tool"
)

var cmdWrite = &tool.Command{
	UsageLine:    "write device msg...",
	Short:        "write CAN messages",
	Long:         ``,
	ExtraArgsReq: 2,
}

func init() {
	cmdWrite.Flag.UintVar(&id, "id", 0x123, "CAN id")
	cmdWrite.Flag.BoolVar(&extFrameFlag, "ext", false, "set extended frame flag")

	cmdWrite.Run = runWrite
}

var id uint
var extFrameFlag bool

func runWrite(cmd *tool.Command, w io.Writer, args []string) (err error) {
	devName := args[0]
	args = args[1:]

	dev, err := can.Open(devName)
	if err != nil {
		return err
	}
	defer dev.Close()

	var m can.Msg
	for _, spec := range args {
		m.Reset()
		m.Id = uint32(id)
		if extFrameFlag {
			m.Flags = can.ExtFrame
		}
		err := m.FromExpr(spec)
		if err != nil {
			return err
		}
		err = dev.WriteMsg(&m)
		if err != nil {
			return err
		}
	}
	return nil
}
