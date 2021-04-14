// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pcan

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

const pattern = `*n -type-`

func parseProcfile() (list busList, err error) {
	var index = make(map[string][]channel, 5)

	f, err := os.Open("/proc/pcan")
	if err != nil {
		return
	}
	defer f.Close()

	r := bufio.NewReader(f)
	skip := true
	for {
		line, e := r.ReadString('\n')
		if e != nil {
			break
		}
		if strings.HasPrefix(line, pattern) {
			skip = false
		}
		if skip || strings.HasPrefix(line, "*") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		minor, e := strconv.Atoi(f[0])
		if e != nil {
			continue
		}

		netdev := f[2]
		if netdev == "-NA-" {
			netdev = ""
		}

		typ := f[1]
		switch typ {
		case "pccard":
			typ = "pcc"
		case "usbpro":
			typ = "usb"
		}
		index[typ] = append(index[typ], channel{minor, netdev})
	}

	list = []*bus{
		{"usb", nil},
		{"usbfd", nil},
		{"pci", nil},
		{"pcc", nil},
		{"dng", nil},
		{"isa", nil},
	}

	for _, b := range list {
		if c, ok := index[b.name]; ok {
			b.channels = c
		}
	}
	return
}
