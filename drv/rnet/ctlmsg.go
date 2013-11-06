// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rnet

import (
	"encoding/binary"
	"io"
)

const (
	hdrIDctl = 0x10
)

const (
	// cmd flags
	cmdFifoActive = 1 << iota
	cmdListenTcp
)

const (
	// ctl flags
	ctlClientQuiet = 1 << iota
	ctlClientReliable
	ctlClientOmniReader
	ctlClientOmniWriter
)

type ctlMsg struct {
	Len         uint16
	Type        byte
	Cmd, Client struct {
		Present, Flags byte
	}
}

func (m *ctlMsg) WriteTo(w io.Writer) (n int64, err error) {
	m.Len = 2 + 1 + 4
	m.Type = hdrIDctl
	err = binary.Write(w, bo, m)
	n = int64(binary.Size(m))
	return
}
