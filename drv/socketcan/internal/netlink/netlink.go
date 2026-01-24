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
	"github.com/knieriem/can"
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

type Conn struct {
	*rtnetlink.Conn
}

var ErrNotFound = errors.New("not found")

type Interface struct {
	Name      string
	conn      *Conn
	index     int
	msgFamily uint16
	msgType   uint16
}

func Dial() (*Conn, error) {
	// Dial a connection to the rtnetlink socket
	conn, err := rtnetlink.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("netlink: %w", err)
	}
	return &Conn{Conn: conn}, nil
}

func (conn *Conn) OpenInterface(name string) (*Interface, error) {
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

func (intf *Interface) UpDown(up bool) error {
	link, err := intf.get()
	if err != nil {
		return err
	}

	var flags uint32
	st := link.Attributes.OperationalState
	if up {
		if st == rtnetlink.OperStateUp || st == rtnetlink.OperStateUnknown {
			return nil
		}
		flags = unix.IFF_UP
	} else if st == rtnetlink.OperStateDown {
		return nil
	}

	return intf.conn.Link.Set(&rtnetlink.LinkMessage{
		Family: link.Family,
		Type:   link.Type,
		Index:  uint32(intf.index),
		Flags:  flags,
		Change: unix.IFF_UP,
	})
}

func (intf *Interface) SetConfig(conf *can.Config) error {
	enc := NewCANAttrEncoder()
	enc.SetConfig(conf)
	return enc.UpdateLink(intf)

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

func (link *Link) NeedUpdate(conf *can.Config) (needUpdate bool, err error) {
	needUpdate, err = link.Can.needUpdate(conf)
	if err != nil {
		return false, err
	}
	if needUpdate {
		return true, nil
	}
	return link.Attr.OperationalState != rtnetlink.OperStateUp, nil
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
