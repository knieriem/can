// Package socketcan implements a [github.com/knieriem/can.Driver]
// for SocketCAN on Linux.
//
// Interfaces are configured through the kernel's netlink interface.
// Changing an interface's bit timing needs CAP_NET_ADMIN;
// [WithPrivilegedUtil] runs that part in a separate, privileged helper.
//
// # Receive buffer
//
// The kernel silently drops received frames when a socket's receive
// buffer is full, for example while the reading program is not
// scheduled for a moment at high bus load. The driver therefore
// requests a buffer of 1 MiB. Without CAP_NET_ADMIN, the kernel caps
// it at net.core.rmem_max, about 208 KiB by default on most systems.
// To give unprivileged programs the full size, raise the limit,
// for example in /etc/sysctl.d/:
//
//	net.core.rmem_max = 1048576
package socketcan
