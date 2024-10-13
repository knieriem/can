// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"fmt"
	"syscall"
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

// Copied from syscall/syscall_unix.go

// Do the interface allocations only once for common
// Errno values.
var (
	errEAGAIN error = syscall.EAGAIN
	errEINVAL error = syscall.EINVAL
	errENOENT error = syscall.ENOENT
)

// errnoErr returns common boxed Errno values, to prevent
// allocations at runtime.
func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case syscall.EAGAIN:
		return errEAGAIN
	case syscall.EINVAL:
		return errEINVAL
	case syscall.ENOENT:
		return errENOENT
	}
	return e
}
