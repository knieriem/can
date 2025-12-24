// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zapi_windows.go api_windows.go
package api

type Status uint32

func (s Status) Err() (err error) {
	if s != OK {
		err = s
	}
	return
}

func (st Status) Test(flags Status) bool {
	return st&flags != 0
}

func bufToString(buf []byte) (s string) {
	for i, b := range buf {
		if b == 0 {
			buf = buf[:i]
			break
		}
	}
	s = string(buf)
	return
}
