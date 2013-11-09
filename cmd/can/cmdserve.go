// Copyright 2013 The hgo Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"log"
	"net"
	"net/rpc"

	"can"
	_ "can/drv/can4linux"
	"can/drv/canrpc"
	_ "can/drv/pcan"
	_ "can/drv/rnet"
)

var cmdServe = &Command{
	UsageLine: "serve [-t] addr [device]",
	Short:     "serve a CAN device on a tcp port",
	Long:      ``,
}

func init() {
	//	addStdFlags(cmdServe)
	cmdServe.Run = runServe
}

func runServe(cmd *Command, w io.Writer, args []string) {
	if len(args) < 1 {
		usage()
	}

	devName := ""
	if len(args) > 1 {
		devName = args[1]
	}
	d, err := can.Open(devName)
	if err != nil {
		log.Fatal(err)
	}
	err = canrpc.Register(rpc.DefaultServer, d, "")
	l, err := net.Listen("tcp", args[0])
	if err != nil {
		return
	}
	log.Println("Listening on", l.Addr(), "...")
	rpc.Accept(l)
	return
}
