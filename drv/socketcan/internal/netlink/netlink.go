package netlink

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/jsimonetti/rtnetlink"
)

func List() ([]*Link, error) {
	conn, err := rtnetlink.Dial(nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	msg, err := conn.Link.List()
	if err != nil {
		return nil, err
	}

	list := make([]*Link, 0, 8)

	for i := range msg {
		link := &msg[i]
		if link.Type != unix.ARPHRD_CAN {
			continue
		}
		info, err := newLinkInfo(link.Attributes)
		if err != nil {
			continue
		}
		list = append(list, info)
	}
	if len(list) == 0 {
		return nil, nil
	}
	sort.Sort(linkList(list))
	return list, nil
}

type linkList []*Link

func (list linkList) Len() int      { return len(list) }
func (list linkList) Swap(i, j int) { list[i], list[j] = list[j], list[i] }
func (list linkList) Less(i, j int) bool {
	a := list[i]
	b := list[j]
	pfxa, ia := splitName(a.Name())
	pfxb, ib := splitName(b.Name())
	if pfxa < pfxb {
		return true
	}
	if pfxa > pfxb {
		return false
	}
	return ia < ib
}

func splitName(name string) (prefix string, index int) {
	i := strings.IndexAny(name, "0123456789")
	if i == -1 {
		return name, -1
	}
	prefix = name[:i]
	index, err := strconv.Atoi(prefix[i:])
	if err != nil {
		return prefix, math.MaxInt
	}
	return prefix, index
}

var ErrNotFound = errors.New("not found")

type Interface struct {
	Name      string
	conn      *rtnetlink.Conn
	index     int
	msgFamily uint16
	msgType   uint16
}

func OpenInterface(name string) (*Interface, error) {
	if name == "" {
		list, err := List()
		if err != nil {
			return nil, err
		}
		if list == nil {
			return nil, ErrNotFound
		}
		name = list[0].Name()
	}
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return nil, err
	}

	// Dial a connection to the rtnetlink socket
	conn, err := rtnetlink.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("netlink: %w", err)
	}

	l := new(Interface)
	l.Name = name
	l.conn = conn
	l.index = ifi.Index
	msg, err := l.get()
	if err != nil {
		conn.Close()
		return nil, err
	}
	l.msgFamily = msg.Family
	l.msgType = msg.Type
	return l, nil
}

func (link *Interface) Close() error {
	return link.conn.Close()
}

func (link *Interface) get() (*rtnetlink.LinkMessage, error) {
	msg, err := link.conn.Link.Get(uint32(link.index))
	if err != nil {
		return nil, fmt.Errorf("netlink: %w", err)
	}

	if msg.Type != unix.ARPHRD_CAN {
		return nil, fmt.Errorf("netlink: not a can device: %q", link.Name)
	}
	return &msg, nil
}

func (link *Interface) Info() (*Link, error) {
	msg, err := link.get()
	if err != nil {
		return nil, err
	}
	info, err := newLinkInfo(msg.Attributes)
	if err != nil {
		return nil, fmt.Errorf("netlink: %w", err)
	}

	return info, nil
}

type Link struct {
	Attr *rtnetlink.LinkAttributes
	Can  *CanAttributes
}

func newLinkInfo(a *rtnetlink.LinkAttributes) (*Link, error) {
	link := new(Link)
	link.Attr = a
	if a.Info == nil || a.Info.Kind != "can" {
		return link, nil
	}
	ca := new(CanAttributes)
	ca, err := decodeCanAttributes(a.Info.Data)
	if err != nil {
		return nil, err
	}
	link.Can = ca
	return link, nil
}

func (link *Link) Name() string {
	return link.Attr.Name
}

func (link *Link) DriverName() string {
	if link.Can == nil {
		if info := link.Attr.Info; info != nil && info.Kind == "vcan" {
			return "vcan"
		}
		return ""
	}
	btc := link.Can.BitTimingConst
	if btc == nil {
		return ""
	}
	b := btc.Name[:]
	if i := bytes.IndexByte(b, 0); i != -1 {
		b = b[:i]
	}
	return string(b)
}

func (link *Link) Bitrate() (uint32, bool) {
	if link.Can == nil {
		return 0, false
	}
	if link.Can.BitTiming == nil {
		return 0, false
	}
	return link.Can.BitTiming.Bitrate, true
}
