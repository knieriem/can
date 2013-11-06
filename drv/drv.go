// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// A helper package for the various CAN driver interface packages.
package drv

import "can"

type FlagsMap []struct {
	Pkg can.Flags // a flag as defined by this package
	Drv int       // a flag as defined by the driver
}

func (m FlagsMap) Decode(v int) (f can.Flags) {
	for i := range m {
		if v&m[i].Drv != 0 {
			f |= can.Flags(m[i].Pkg)
		}
	}
	return
}
func (m FlagsMap) Encode(f can.Flags) (v int) {
	for i := range m {
		if f&m[i].Pkg != 0 {
			v |= m[i].Drv
		}
	}
	return
}
