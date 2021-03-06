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
	_ "github.com/knieriem/can/drv/can4linux"
	"github.com/knieriem/can/drv/canrpc"
	_ "github.com/knieriem/can/drv/pcan"

	"github.com/knieriem/tool"
)

var cmdServe = &tool.Command{
	UsageLine:    "serve [-t] addr [device]",
	Short:        "serve a CAN device on a tcp port",
	Long:         ``,
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
	d, err := can.Open(devName)
	if err != nil {
		return
	}
	err = canrpc.Register(rpc.DefaultServer, d, "")
	l, err := net.Listen("tcp", args[0])
	if err != nil {
		return
	}
	fmt.Println("Listening on", l.Addr(), "...")
	rpc.Accept(l)
	return
}
