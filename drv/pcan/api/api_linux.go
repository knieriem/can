// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"fmt"
)

const (
	ErrANYBUSERR Status = (ErrBUSLIGHT | ErrBUSHEAVY | ErrBUSOFF)
)

type Fd uintptr

func (f Fd) Status() (s Status) {
	var p statusPar

	if err := f.status(&p); err != nil {
		s = ErrNOVXD
	} else {
		s = Status(p.WErrorFlag)
	}
	return
}

func (s Status) Error() string {
	return fmt.Sprintf("status: 0x%04X", uint32(s))
}

//sys	ioctl(fd uintptr, action uintptr, arg uintptr) (err error) = SYS_IOCTL

func (d *Diag) Version() string {
	return bufToString(d.SzVersionString[:])
}
