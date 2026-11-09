package main

import (
	"fmt"
	"io"

	"github.com/knieriem/can"
	"github.com/knieriem/tool"
)

var cmdTrace = &tool.Command{
	UsageLine: "trace [-e] device",
	Short:     "log CAN messages",
	Long: `
Trace prints received frames and status messages.
`,
	ExtraArgsMax: 1,
}

var traceBusErrors bool

func init() {
	cmdTrace.Run = runTrace
	cmdTrace.Flag.BoolVar(&traceBusErrors, "e", false, "also print a status message for every bus error")
}

func runTrace(cmd *tool.Command, w io.Writer, args []string) (err error) {
	devName := ""
	if len(args) > 0 {
		devName = args[0]
	}

	opt := can.WithStatusMsgs()
	if traceBusErrors {
		opt = can.WithBusErrors()
	}
	dev, err := can.Open(devName, opt)
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
