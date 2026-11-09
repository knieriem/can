package main

import (
	"github.com/knieriem/tool"

	"github.com/knieriem/can/drv/all"
)

func main() {
	all.Register()
	tool.Name = "can"
	tool.Title = "CAN utility."
	tool.Version = "0.1"
	tool.Commands = []*tool.Command{
		tool.CmdVersion,
		cmdLsDev,
		cmdWrite,
		cmdTrace,
		cmdBittiming,
		cmdServe,
		cmdExpose,
	}
	tool.Run()
}
