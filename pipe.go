// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can

import (
	"io"
)

type pipeDev struct {
	r, w chan Msg
	Unversioned
}

func NewPipe() (Device, Device) {
	c1 := make(chan Msg, 16)
	c2 := make(chan Msg, 16)
	return &pipeDev{r: c1, w: c2}, &pipeDev{r: c2, w: c1}
}

func (d *pipeDev) Read(buf []Msg) (n int, err error) {
	var ok bool

	if len(buf) == 0 {
		return
	}
	buf[0], ok = <-d.r
	if !ok {
		err = io.EOF
		close(d.w)
	} else {
		n = 1
	}
	return
}

func (d *pipeDev) Write(msgs []Msg) (n int, err error) {
	for i := range msgs {
		d.w <- msgs[i]
		n++
	}
	return
}
func (d *pipeDev) WriteMsg(m *Msg) (err error) {
	d.w <- *m
	return
}

func (d *pipeDev) Close() (err error) {
	close(d.w)
	return
}
