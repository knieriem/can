package main

import (
	"fmt"
	"io"

	"github.com/knieriem/can"
	"github.com/knieriem/tool"
)

var cmdLsDev = &tool.Command{
	ExtraArgsMax: 1,
	UsageLine:    "lsdev",
	Short:        "list CAN devices",
	Run:          runLsDev,
}

func runLsDev(cmd *tool.Command, w io.Writer, args []string) error {
	for _, name := range can.Scan() {
		fmt.Print(name.Format("\t(", ", ", ")\n"))
	}
	return nil
}
