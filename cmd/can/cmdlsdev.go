package main

import (
	"fmt"
	"io"

	"can"
)

var cmdLsDev = &Command{
	ExtraArgsMax: 1,
	UsageLine:    "lsdev",
	Short:        "list serial and CAN devices",
	Run:          runLsDev,
}

func runLsDev(cmd *Command, w io.Writer, args []string) error {
	for _, name := range can.Scan() {
		fmt.Print(name.Format("\t(", ", ", ")\n"))
	}
	return nil
}
