// Copyright 2013 The hgo Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"net"
	"net/rpc"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/canrpc"
	"github.com/knieriem/tool"
)

var cmdServe = &tool.Command{
	UsageLine: "serve addr [device]",
	Short:     "serve a CAN device on a tcp port",
	Long: `
Serve makes the CAN device available to canrpc clients at addr.
It has no authentication: bind it to localhost, for example
127.0.0.1:6000, and reach it through an SSH tunnel.
`,
	ExtraArgsReq: 1,
	ExtraArgsMax: 2,
}

func init() {
	//	addStdFlags(cmdServe)
	cmdServe.Run = runServe
}

func runServe(cmd *tool.Command, w io.Writer, args []string) (err error) {
	devName := ""
	if len(args) > 1 {
		devName = args[1]
	}
	d, err := can.Open(devName, can.WithBusErrors()) // clients filter themselves
	if err != nil {
		return
	}
	err = canrpc.Register(rpc.DefaultServer, d, "")
	if err != nil {
		return
	}
	l, err := net.Listen("tcp", args[0])
	if err != nil {
		return
	}
	fmt.Println("Listening on", l.Addr(), "...")
	rpc.Accept(l)
	return
}
