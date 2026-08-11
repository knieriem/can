package main

import (
	"fmt"
	"io"

	"github.com/knieriem/can"
	"github.com/knieriem/tool"
)

var cmdTrace = &tool.Command{
	UsageLine:    "trace device",
	Short:        "log CAN messages",
	Long:         ``,
	ExtraArgsMax: 1,
}

func init() {
	cmdTrace.Run = runTrace
}

func runTrace(cmd *tool.Command, w io.Writer, args []string) (err error) {
	devName := ""
	if len(args) > 0 {
		devName = args[0]
	}

	dev, err := can.Open(devName)
	if err != nil {
		return err
	}
	defer dev.Close()

	var buf [64]can.Msg
	for {
		n, err := dev.Read(buf[:])
		if err != nil {
			return err
		}
		for i := range buf[:n] {
			m := &buf[i]
			s := m.Format()
			fmt.Println(s)
		}
	}
}
