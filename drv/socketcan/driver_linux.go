//go:generate sh internal/linux/gen.sh internal/linux
//go:build linux

package socketcan

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/socketcan/internal/linux"
	"github.com/knieriem/can/drv/socketcan/internal/netlink"
)

var Driver = defaultDriver

var defaultDriver = can.Driver(&driver{
	privilegedCmd: "socketcan-link",
})

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
	file io.ReadWriteCloser
	mtu  int
	info can.DeviceInfo

	sendBufMu sync.Mutex
	sendBuf   frame

	recvBuf frame
	receive struct {
		t0    can.Time
		t0val int64
	}
}

func (d *dev) ID() string {
	return "socketcan:" + d.info.ID
}

func (driver) Scan() []can.DeviceInfo {
	linkList, err := netlink.List()
	if err != nil {
		return nil
	}
	if len(linkList) == 0 {
		return nil
	}

	list := make([]can.DeviceInfo, len(linkList))
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
		info, err := link.Info()
		if err != nil {
			return nil, err
		}
		if info.Can == nil {
			if info.IsVCAN() {
				goto skipConfig
			}
			return nil, errors.New("netlink: unexpected nil CAN attributes")
		}
		ctl := info.Can.Controller()
		fd, err := conf.ResolveFDMode(ctl.Data != nil)
		if err != nil {
			return nil, err
		}
		if !fd {
			conf.FDMode.Valid = false
			conf.Data.Valid = false
		}
		err = conf.ResolveBitTiming(ctl)
		if err != nil {
			return nil, err
		}

		// avoid bitrates being printed when formatting bit timings during config
		conf.Nominal.Bitrate = 0
		conf.Nominal.BitrateMult = 0
		conf.Data.Value.Bitrate = 0
		conf.Data.Value.BitrateMult = 0

		needUpdate, err := info.NeedUpdate(conf)
		if err != nil {
			return nil, err
		}

		if needUpdate {
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
	}
skipConfig:

	if devName == "" {
		devName = link.Name
	}

	fd, err := setupSocket(devName)
	if err != nil {
		return nil, wrapErr("open", err)
	}
	ok := false
	defer func() {
		if !ok {
			unix.Close(fd)
		}
	}()
	setRcvBuf(fd)

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
		return nil, wrapErr("open", fmt.Errorf("cannot enter FD mode: %w", err))
	}

	if conf != nil {
		if filters := conf.MsgFilter; len(filters) > 0 {
			flt := make([]unix.CanFilter, len(filters))
			for i := range filters {
				f := &filters[i]

				id := f.ID
				if f.Invert {
					id |= unix.CAN_INV_FILTER
				}
				if f.ExtFrame {
					id |= unix.CAN_EFF_FLAG
				}
				flt[i] = unix.CanFilter{
					Id:   id,
					Mask: f.IDMask | unix.CAN_EFF_FLAG | unix.CAN_RTR_FLAG,
				}
			}
			err := unix.SetsockoptCanRawFilter(fd, unix.SOL_CAN_RAW, unix.CAN_RAW_FILTER, flt)
			if err != nil {
				return nil, wrapErr("open", err)
			}
		}
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
	setupInfo(&d.info, info)
	cleanupPriv = nil
	ok = true
	return d, nil
}

// setupSocket is setting up a raw CAN socket, as described in
// https://www.kernel.org/doc/html/latest/networking/can.html#how-to-use-socketcan
func setupSocket(dev string) (int, error) {
	fd, err := unix.Socket(unix.AF_CAN, unix.SOCK_RAW, unix.CAN_RAW)
	if err != nil {
		return -1, err
	}
	ifr, err := unix.NewIfreq(dev)
	if err == nil {
		err = unix.IoctlIfreq(fd, unix.SIOCGIFINDEX, ifr)
	}
	if err == nil {
		err = unix.Bind(fd, &unix.SockaddrCAN{Ifindex: int(ifr.Uint32())})
	}
	if err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

// rcvBufSize is the socket receive buffer the driver requests.
// At 500 kbit/s it holds roughly a second of frames.
const rcvBufSize = 1 << 20

// setRcvBuf enlarges the socket receive buffer to rcvBufSize, but
// never shrinks it, as a system may have a larger rmem_default.
// Errors are ignored: the default buffer still works.
func setRcvBuf(fd int) {
	cur, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF)
	if err != nil || cur >= rcvBufSize {
		return
	}
	if unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUFFORCE, rcvBufSize) == nil {
		return
	}
	// Without CAP_NET_ADMIN, the kernel caps the request at rmem_max,
	// and doubles it. Request it only if the result is larger.
	max, err := rmemMax()
	if err != nil || 2*min(rcvBufSize, max) <= cur {
		return
	}
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, rcvBufSize)
}

func rmemMax() (int, error) {
	b, err := os.ReadFile("/proc/sys/net/core/rmem_max")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}

// pollableFile turns fd into an *os.File, that is managed by Go's runtime poller
func pollableFile(fd int) (io.ReadWriteCloser, error) {
	if err := unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "socket"), nil
}

func setupInfo(info *can.DeviceInfo, link *netlink.Link) {
	*info = can.DeviceInfo{
		ID:           link.Attr.Name,
		Device:       link.Attr.Name,
		Driver:       "socketcan",
		SystemDriver: link.DriverName(),
	}
}

func (d *dev) Read(buf []can.Msg) (n int, err error) {
	f := d.recvBuf
	nRead, err := f.readFromN(d.file, d.mtu)
	if err != nil {
		return 0, wrapErr("read", closedErr(err))
	}
	f.decode(&buf[0], nRead)
	return 1, nil
}

func (d *dev) WriteMsg(msg *can.Msg) error {

	if msg.IsStatus() {
		return nil
	}

	d.sendBufMu.Lock()
	defer d.sendBufMu.Unlock()
	f := d.sendBuf

	nf, err := f.encode(msg, d.mtu)
	if err != nil {
		return wrapErr("write", err)
	}
	_, err = d.file.Write(f.b[:nf])
	if err != nil {
		if errors.Is(err, syscall.ENOBUFS) {
			err = can.ErrTxQueueFull
		}
		return wrapErr("write", closedErr(err))
	}
	return nil
}

func (d *dev) Close() error {
	err := d.file.Close()
	if err != nil {
		return wrapErr("close", closedErr(err))
	}
	return nil
}

func (d *dev) Info() *can.DeviceInfo {
	return &d.info
}

// closedErr translates the error of an operation on a closed file
// into can.ErrClosed.
func closedErr(err error) error {
	if errors.Is(err, os.ErrClosed) {
		return can.ErrClosed
	}
	return err
}

func wrapErr(fnName string, err error) error {
	return fmt.Errorf("socketcan: %s: %w", fnName, err)
}
