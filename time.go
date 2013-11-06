// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package can

import (
	"time"
)

type Time int64

func UnixTimevals(s, µs int32) Time {
	return Time(s)*1e6 + Time(µs)
}

func (t Time) UnixTimevals() (s, µs int32) {
	s = int32(t / 1e6)
	µs = int32(t % 1e6)
	return
}

func (t Time) unixNano() (sec, nsec int64) {
	sec = int64(t) / 1e6
	nsec = (int64(t) % 1e6) * 1000
	return
}

func (t Time) Time() time.Time {
	return time.Unix(t.unixNano())
}

func Now() Time {
	return Time(time.Now().UnixNano() / 1000)
}
