package main

import (
	"fmt"
	"io"
)

var cmdVersion = &Command{
	UsageLine: "version",
	Short:     "displays the program version",
	Run: func(_ *Command, _ io.Writer, _ []string) error {
		fmt.Println("can version 0.1")
		return nil
	},
}
