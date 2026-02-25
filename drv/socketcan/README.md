# socketcan

Package *socketcan* implements a can.Driver for [SocketCAN] on `linux/{amd64,arm,arm64}`.

CAN interface configuration is done using the kernel's [Netlink] interface,
with help of [github.com/mdlayher/netlink] and [github.com/jsimonetti/rtnetlink].


[SocketCAN]: https://docs.kernel.org/networking/can.html
[Netlink]: https://en.wikipedia.org/wiki/Netlink
[github.com/mdlayher/netlink]: https://github.com/mdlayher/netlink
[github.com/jsimonetti/rtnetlink]: https://github.com/jsimonetti/rtnetlink
