package main

import (
	"github.com/knieriem/can"
	"github.com/knieriem/tool"

	_ "github.com/knieriem/can/drv/canrpc"
	_ "github.com/knieriem/can/drv/pcan"
	"github.com/knieriem/can/drv/socketcan"
)

func init() {
	drv := socketcan.NewDriver(socketcan.WithPrivilegedUtil())
	can.RegisterDriver(drv)
}

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
