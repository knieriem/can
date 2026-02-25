# Go CAN Networking Package

Package *can* aims to provide a generic way to access CAN networks
on various platforms.

It has been created in 2013 to enable tools written in Go to
access PCAN USB adapters on both Linux and Windows,
replacing earlier C-based work from a decade before
that focused on can4linux.  
Starting 2022, CAN FD and SocketCAN support was added gradually.  

There is still no v1, due to its age the package has undergone multiple
refactorings.
Error handling is not optimal,
it should align more to what SocketCAN defines.

Support for specific types of adapters is implemented as [Driver]s in
subdirectory `drv`:

| Driver    | Adapters            | Platforms      | CAN 2.0 | FD            |
|-----------|---------------------|----------------|:-------:|---------------|
| pcan      | PCAN-USB            | Linux, Windows |    ☑    | —             |
| pcan      | PCAN-USB FD         | Linux*, Windows |    ☑    | ☑ (Windows) |
|           |
| socketcan | any adapter supported by SocketCAN | Linux | ☑ | ☑             |
|           |
| rpc       | remote CAN adapters |                |    ☑    | ☑             |

\* the FD mode of PCAN-USB FD may be used on Linux via the `socketcan` driver,
but not yet via the `pcan` character-device driver.

Windows Support includes the arm64 architecture.

[Driver]: https://pkg.go.dev/github.com/knieriem/can#Driver


## SocketCAN

The `socketcan` driver makes use of a utility `socketcan-link`,
which is part of the repository and expected to be run with elevated privileges,
specifically `CAP_NET_ADMIN` (to allow CAN interface configuration without root).
See [drv/socketcan] for details.

	cd drv/socketcan/cmd/socketcan-link
	go build -o /path/to/bin/socketcan-link -trimpath -ldflags '-s -w'
	sudo setcap cap_net_admin=ep /path/to/bin/socketcan-link

[SocketCAN]: https://docs.kernel.org/networking/can.html
[drv/socketcan]: ./drv/socketcan/README.md

## Device / Interface configuration

Package `can` provides a Plan 9 `ctl` file inspired text based configuration string
that allows to specify the can device (adapter, interface) and to configure the bit timings (see [ParseConfig] documentation for details).

This config string can be as simple as `",500k"`,
which will tell `can.Open()` to look for the first available CAN adapter and configure it using a nominal bitrate of 500  kbit/s.
The comma separates the empty device string from the parameter `"500k"`.  
To look specifically for a PCAN adapter, use `"pcan,500k"`,
which will use the first available PCAN device.
`"pcan:usb2,500k"` narrows the configuration to the second PCAN USB adapter.
Similar, `"socketcan"` will use any available SocketCAN adapter,
while "socketcan:can0" explicitely selects the `can0` network interface,
and "socketcan:@spi0.1" would select the network interface linked to SPI device `0.1`.

On default, a sample point of 87.5% is assumed.
To specify a different sample point, use for instance: `500k@.7` for 70%.
A data bittiming for CAN FD mode can be specified using the `db:` prefix:

	500k@.8,db:1M@.7

This will set the nominal bittiming to 500 kbit/s at 80%,
and the data bittiming to 1 Mbit/s with a sample point set to 70%.

_Sync jump width_ is set to the size of the phase segment 2 on default,
but can be set to a specific value (in tq or as fraction) if needed,
like 4 tq: `500k@.8s4` or `500k@.8:s4`, or 10 percent: `500k@.8:s.1`

Instead of a bitrate a bit timing specification may be used, like:

	*25:34-35-10

which means 1 tq = 25ns, _propSeg_ = 34 tq, _phaseSeg1_ = 35 tq, _phaseSeg2_ = 10 tq.
This will result in a bitrate of 500 kbit/s.
The `*` character signals the multiplicative nature of the tq value;
there is also `/` to specify a clock prescaler, like `/2`.

FD mode is selected automatically if a data bitrate is specified and the adapter supports FD mode.
It can be enforced by specifying `fd`, like in `,1M,fd`.


[ParseConfig]: https://pkg.go.dev/github.com/knieriem/can@v0.3.0-alpha8#ParseConfig


## Basic Usage

```Go
package main

import (
	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/socketcan"
)

func main() {
	can.RegisterDriver(socketcan.Driver)

	dev, err := can.Open("")
	if err != nil {
		// ...
	}

	var m can.Msg

	// Write a standard frame containing one byte, 0x42, with identifier 0xABC.
	m.Id = 0xABC
	data := m.Data()[:1]
	data[0] = 0x42
	m.SetData(data)
	err = dev.WriteMsg(&m)
	if err != nil {
		// ...
	}
}
```

## cmd/can Utility

Command `can` provides functionality like calculating bit timings:

	./can bt -dev candlelightfd 500k

or writing a CAN frame,
similar but less complete compared to what [can-utils]' `cansend` provides.

[can-utils]: https://github.com/linux-can/can-utils

## Tested Adapters

- PCAN-USB, PCAN-USB FD (`socketcan`, `pcan`; Linux, Windows)
- MCP2518FD (`socketcan`; Linux on RPi)
- [candleLight FD] (USB/STM32G0B1, `socketcan`; Linux)

## Use Cases

*Industrial*  
This package has been used for CAN 2.0 tooling,
e.g. as part of a Modbus-over-CAN firmware downloader,
or as development tool.
In these cases, the PCAN-USB adapters are used on Windows and Linux.

*Automotive*  
The package is also used to communicate with LED drivers like ST's LLDL16EN on a CAN FD light bus.
For this application, SocketCAN is used, with a MCP2518FD on linux/arm64.


[candleLight FD]: https://linux-automation.com/en/products/candlelight-fd.html