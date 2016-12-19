package main

import (
	"tool"
)

func main() {
	tool.Name = "can"
	tool.Title = "Can"
	tool.Version = "0.1"
	tool.Commands = []*tool.Command{
		tool.CmdVersion,
		cmdLsDev,
		cmdServe,
	}
	tool.Run()
}
