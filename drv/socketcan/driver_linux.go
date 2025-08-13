//go:generate sh internal/linux/gen.sh internal/linux

package socketcan

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/socketcan/internal/linux"
	"github.com/knieriem/can/drv/socketcan/internal/netlink"
)

func NewDriver(opts ...DriverOption) can.Driver {
	drv := new(driver)
	for _, o := range opts {
		o(drv)
	}
	return drv
}

type DriverOption func(*driver)

func WithPrivilegedUtil() DriverOption {
	return func(d *driver) {
		d.privilegedCmd = "socketcan-link"
	}
}

type driver struct {
	privilegedCmd string
}

func (driver) Name() string {
	return "socketcan"
}

type dev struct {
	file    io.ReadWriteCloser
	mtu     int
	name    can.Name
	recvBuf frame
	receive struct {
		t0    can.Time
		t0val int64
	}
}

func (d *dev) ID() string {
	return "socketcan:" + d.name.ID
}

func (driver) Scan() []can.Name {
	linkList, err := netlink.List()
	if err != nil {
		return nil
	}
	if len(linkList) == 0 {
		return nil
	}

	list := make([]can.Name, len(linkList))
	for i, link := range linkList {
		setupInfo(&list[i], link)
	}
	return list
}

func (drv *driver) Open(devName string, conf *can.Config) (can.Device, error) {
	if strings.HasPrefix(devName, "@") {
		// devName is system device
		dev := devName[1:]
		if strings.HasPrefix(dev, "spi") {
			d, err := os.ReadDir(filepath.Join("/sys/bus/spi/devices", dev, "net"))
			if err != nil {
				return nil, err
			}
			if len(d) == 0 {
				return nil, fmt.Errorf("could not determine network interface for %q", dev)
			}
			devName = d[0].Name()
		} else {
			return nil, fmt.Errorf("device %q not recognized", dev)
		}
	}
	conn, err := netlink.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	link, err := conn.OpenInterface(devName)
	if err != nil {
		return nil, err
	}

	var cleanupPriv func()
	if conf != nil {
		priv := privilegedAccess(&privilegedDirect{Interface: link})
		if drv.privilegedCmd != "" {
			priv, err = startPrivilegedUtil(drv.privilegedCmd, link.Name)
			if err != nil {
				return nil, err
			}
			defer func() {
				if cleanupPriv != nil {
					cleanupPriv()
				}
			}()
			cleanupPriv = func() {
				priv.Close()
			}
		}
		err = priv.SetConfig(conf)
		if err != nil {
			return nil, err
		}
		err = priv.UpDown(true)
		if err != nil {
			return nil, err
		}
	}

	if devName == "" {
		devName = link.Name
	}

	fd, err := setupSocket(devName)
	if err != nil {
		return nil, wrapErr("open", err)
	}

	info, err := link.Info()
	if err != nil {
		return nil, err
	}

	d := new(dev)
	d.mtu = int(info.Attr.MTU)
	err = unix.SetsockoptInt(fd, unix.SOL_CAN_RAW, unix.CAN_RAW_FD_FRAMES, 1)
	if err != nil {
		if d.mtu > linux.CAN_MTU {
			d.mtu = linux.CAN_MTU
		}
		return nil, wrapErr("setsockopt", fmt.Errorf("cannot enter FD mode: %w", err))
	}

	errMask := linux.CAN_ERR_CRTL |
		linux.CAN_ERR_BUSOFF |
		linux.CAN_ERR_ACK |
		linux.CAN_ERR_BUSERROR |
		linux.CAN_ERR_RESTARTED |
		linux.CAN_ERR_TX_TIMEOUT
	err = unix.SetsockoptInt(fd, unix.SOL_CAN_RAW, unix.CAN_RAW_ERR_FILTER, errMask)
	if err != nil {
		return nil, wrapErr("open", err)
	}
	file, err := pollableFile(fd)
	if err != nil {
		return nil, wrapErr("open", err)
	}
	d.file = file
	setupInfo(&d.name, info)
	cleanupPriv = nil
	return d, nil
}

// setupSocket is setting up a raw CAN socket, as described in
// https://www.kernel.org/doc/html/latest/networking/can.html#how-to-use-socketcan
func setupSocket(dev string) (fd int, err error) {
	fd, err = unix.Socket(unix.AF_CAN, unix.SOCK_RAW, unix.CAN_RAW)
	if err != nil {
		return -1, err
	}

	ifr, err := unix.NewIfreq(dev)
	if err != nil {
		return -1, err
	}
	err = unix.IoctlIfreq(fd, unix.SIOCGIFINDEX, ifr)
	if err != nil {
		return -1, err
	}

	err = unix.Bind(fd, &unix.SockaddrCAN{Ifindex: int(ifr.Uint32())})
	if err != nil {
		return -1, err
	}
	return fd, nil
}

// pollableFile turns fd into an *os.File, that is managed by Go's runtime poller
func pollableFile(fd int) (io.ReadWriteCloser, error) {
	if err := unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "socket"), nil
}

func setupInfo(info *can.Name, link *netlink.Link) {
	dev := link.DriverName()
	if dev != "" {
		dev += ":"
		dev += link.Attr.Name
	}
	*info = can.Name{
		ID:     link.Attr.Name,
		Device: dev,
		Driver: "socketcan",
	}
}

func (d *dev) Read(buf []can.Msg) (n int, err error) {
	f := d.recvBuf
	err = f.readFromN(d.file, d.mtu)
	if err != nil {
		return 0, wrapErr("read", err)
	}
	err = f.decode(&buf[0])
	if err != nil {
		return 0, err
	}
	return 1, nil
}

func (d *dev) WriteMsg(msg *can.Msg) error {
	var f frame

	if msg.IsStatus() {
		return nil
	}
	nf, err := f.encode(msg, d.mtu)
	if err != nil {
		return wrapErr("write", err)
	}
	_, err = d.file.Write(f.b[:nf])
	if err != nil {
		return wrapErr("write", err)
	}
	return nil
}

func (d *dev) Write(msgs []can.Msg) (n int, err error) {
	for i := range msgs {
		err = d.WriteMsg(&msgs[i])
		if err != nil {
			break
		}
		n++
	}
	return
}

func (d *dev) Close() error {
	err := d.file.Close()
	if err != nil {
		return wrapErr("close", err)
	}
	return nil
}

func (d *dev) Version() can.Version {
	return can.Version{}
}

func (d *dev) Name() can.Name {
	return d.name
}

func wrapErr(fnName string, err error) error {
	return fmt.Errorf("socketcan: %s: %w", fnName, err)
}
