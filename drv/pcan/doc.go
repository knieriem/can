// Package pcan implements a [github.com/knieriem/can.Driver]
// for PEAK-System PCAN adapters.
//
// On Windows, it uses the PCAN-Basic library (PCANBasic.dll),
// which must be installed. CAN FD is supported.
//
// On Linux, it uses the character device interface of PEAK's
// out-of-tree pcan kernel module, for classic CAN only. This backend
// is deprecated and may be removed in a future version. Use the
// socketcan driver instead, with the peak_usb driver that is part of
// the mainline kernel: it supports CAN FD, needs no module rebuild on
// kernel updates, and the socketcan-link helper configures interfaces
// without root. To switch, blacklist the pcan module, for example in
// /etc/modprobe.d/, and remove a blacklist entry for peak_usb, if
// PEAK's installer added one:
//
//	blacklist pcan
package pcan
