# socketcan

Package *socketcan* implements a can.Driver for [SocketCAN] on `linux/{amd64,arm,arm64}`.

Currently, reading and writing of CAN and CAN FD messages is supported.

It is also possible to get a list of CAN interfaces present on a system. This is done
using the kernel's [Netlink] interface, with help of [github.com/mdlayher/netlink].

Interface configuration will be added later in a generic way;
currently it still must be done using a tool like `ip`,
for example:

```sh
ip link set can0 up type can bitrate 500000
```

## Usage

```Go
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
	// use dev
}
```

[SocketCAN]: https://docs.kernel.org/networking/can.html
[Netlink]: https://en.wikipedia.org/wiki/Netlink
[github.com/mdlayher/netlink]: https://github.com/mdlayher/netlink
