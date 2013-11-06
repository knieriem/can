// Created by cgo -godefs - DO NOT EDIT
// cgo -godefs windows/types.go

package api

type MsgType uint8
type HwType uint8
type Baudrate uint16
type Handle uint8
type Mode uint8

type Msg struct {
	ID        uint32
	MSGTYPE   uint8
	LEN       uint8
	DATA      [8]uint8
	Pad_cgo_0 [2]byte
}

type TimeStamp struct {
	Millis   uint32
	Overflow uint16
	Micros   uint16
}
