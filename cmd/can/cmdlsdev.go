package main

import (
	"fmt"
	"io"

	"can"
	"tool"
)

var cmdLsDev = &tool.Command{
	ExtraArgsMax: 1,
	UsageLine:    "lsdev",
	Short:        "list serial and CAN devices",
	Run:          runLsDev,
}

func runLsDev(cmd *tool.Command, w io.Writer, args []string) error {
	for _, name := range can.Scan() {
		fmt.Print(name.Format("\t(", ", ", ")\n"))
	}
	return nil
}
