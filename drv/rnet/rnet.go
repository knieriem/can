// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rnet

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"can"
	"can/drv/can4linux"
	"ibt/mbox"
)

const (
	defaultPort = ":4712"
	maxMsgSize  = 25
)

var bo = binary.BigEndian

func putInt32(b []byte, v int32) {
	bo.PutUint32(b, uint32(v))
}

type dev struct {
	rx *bufio.Reader
	tx struct {
		tmp []byte
		*bufio.Writer
	}
	io.Closer
	name string
	addr string
	can.Unversioned
}

func (d *dev) ID() string {
	return "rnet:" + d.name
}

func (d *dev) Name() (n can.Name) {
	n.ID = d.name
	n.Device = d.addr
	n.Driver = "rnet"
	return
}

func init() {
	can.RegisterDriver(new(driver))
}

type driver struct {
}

func (*driver) Name() string {
	return "rnet"
}

var unixsocket = mbox.Unsharp("#R/rudinet_sock")

func (*driver) Scan() (list []can.Name) {
	fi, err := os.Stat(unixsocket)
	if err == nil && fi.Mode()&os.ModeSocket != 0 {
		list = append(list, can.Name{
			Device: unixsocket,
			Driver: "rnet",
		})
	}
	return
}

func enableRemotePort(host string) (enabled bool) {
	resp, err := http.Get("http://" + host + "/m-box/rudinet.cgi?listen")
	if err == nil {
		resp.Body.Close()
		enabled = true
	}
	return
}

func NewDevice(conn io.ReadWriter) (cd can.Device) {
	cd = newDev(conn, ioutil.NopCloser(conn))
	return
}

func newDev(conn io.ReadWriter, c io.Closer) (d *dev) {
	d = new(dev)
	d.rx = bufio.NewReader(conn)
	d.tx.tmp = make([]byte, maxMsgSize)
	d.tx.Writer = bufio.NewWriter(conn)
	d.Closer = c
	return
}

func (*driver) Open(devName string, _ ...interface{}) (d can.Device, err error) {
	var conn net.Conn
	var addr string

	if devName == "" {
		conn, err = net.Dial("unix", unixsocket)
		addr = unixsocket
	} else {
		addr = devName
		host, _, e := net.SplitHostPort(addr)
		if e != nil {
			host = addr
			addr += defaultPort
		}
		conn, err = net.Dial("tcp", addr)
		if connRefused(err) {
			if ok := enableRemotePort(host); ok {
				conn, err = net.Dial("tcp", addr)
			}
		}
	}
	if err != nil {
		return
	}
	cd := newDev(conn, conn)
	cd.name = devName
	cd.addr = addr

	var c ctlMsg
	c.Client.Present |= ctlClientOmniReader | ctlClientOmniWriter
	c.Client.Flags = c.Client.Present
	c.WriteTo(conn)

	d = cd
	return
}

func (d *dev) Read(buf []can.Msg) (n int, err error) {
	var tmpBuf = make([]byte, maxMsgSize)

	for ; n < len(buf); n++ {
		stillBuffered := true
		if d.rx.Buffered() < 2 {
			stillBuffered = false
			// not enough bytes available in the current buffer
			if n > 0 {
				// already have at least one message, avoid
				// filling the buffer a second time
				return
			}
		}
		avail, err1 := d.rx.Peek(2)
		if err1 != nil {
			err = err1
			return
		}
		//		if _, err = io.ReadFull(d.rx, tmpBuf[:2]); err != nil {
		//			return
		//		}
		msgLen := int(bo.Uint16(avail))

		if msgLen < 2 || msgLen > maxMsgSize {
			err = errors.New("rnet: corrupt message length value")
			return
		}
		//		msgLen -= 2
		if stillBuffered {
			if d.rx.Buffered() < msgLen {
				if n > 0 {
					return
				}
			}
		}
		if _, err = io.ReadFull(d.rx, tmpBuf[:msgLen]); err != nil {
			return
		}
		switch tmpBuf[2] {
		default:
			err = decode(&buf[n], tmpBuf[2:msgLen])
		case hdrIDctl:
		}
	}
	return
}

func (d *dev) Write(msgs []can.Msg) (n int, err error) {
	for i := range msgs {
		sz := encode(d.tx.tmp, &msgs[i])
		d.tx.Write(d.tx.tmp[:sz])
		n++
	}
	d.tx.Flush()
	return
}
func (d *dev) WriteMsg(m *can.Msg) (err error) {
	sz := encode(d.tx.tmp, m)
	d.tx.Write(d.tx.tmp[:sz])
	d.tx.Flush()
	return
}

const (
	hdrNoTimestamp = 0x20 + iota
	hdrDelay
	hdrTimestamp
)

func decode(m *can.Msg, buf []byte) (err error) {
	switch buf[0] {
	default:
		return errors.New("rnet: unknown header id")

	case hdrTimestamp:
		m.Rx.Time = can.UnixTimevals(int32(bo.Uint32(buf[1:])), int32(bo.Uint32(buf[5:])))
		buf = buf[8:]

	case hdrDelay:
		m.Tx.Delayµs = int(bo.Uint16(buf[1:]))
		buf = buf[2:]

	case hdrNoTimestamp:
	}

	f := int(buf[1])
	id := bo.Uint32(buf[2:])
	m.Flags = can4linux.MsgFlagsMap.Decode(f)
	if id == 0xFFFFFFFF && f&can4linux.ErrMask != 0 {
		m.Id = 0
		m.Flags = can4linux.ErrFlagsMap.Decode(f)
	} else {
		m.Id = id
	}

	m.Len = int(buf[6])
	if m.Len > 0 {
		if m.Len != len(buf[7:]) {
			err = errors.New("rnet: corrupted message length")
		} else {
			copy(m.Data[:], buf[7:])
		}
	}
	return
}

func encode(buf []byte, m *can.Msg) (size int) {
	hdrSize := 3
	switch {
	case m.Tx.Delayµs != 0:
		buf[2] = hdrDelay
		bo.PutUint16(buf[3:], uint16(m.Tx.Delayµs))
		hdrSize += 2
	case m.Rx.Time == 0:
		buf[2] = hdrNoTimestamp
	default:
		buf[2] = hdrTimestamp
		sec, µsec := m.Rx.Time.UnixTimevals()
		putInt32(buf[3:], sec)
		putInt32(buf[7:], µsec)
		hdrSize += 8
	}

	if m.Len > 8 {
		m.Len = 8
	}
	size = hdrSize + 1 + 4 + 1 + m.Len
	bo.PutUint16(buf[0:], uint16(size))
	buf = buf[hdrSize-1:]

	f := can4linux.MsgFlagsMap.Encode(m.Flags)
	f |= can4linux.ErrFlagsMap.Encode(m.Flags)
	buf[1] = byte(f)

	id := m.Id
	if m.IsStatus() {
		id = 0xFFFFFFFF
	}
	bo.PutUint32(buf[2:], id)

	buf[6] = byte(m.Len)
	if m.Len > 0 {
		copy(buf[7:], m.Data[:])
	}
	return
}

func connRefused(err error) (is bool) {
	if err == nil {
		return
	}
	for _, pattern := range []string{"refused", "verweigert"} {
		if strings.Contains(err.Error(), pattern) {
			is = true
			break
		}
	}
	return
}
