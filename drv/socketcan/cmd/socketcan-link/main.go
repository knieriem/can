package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/knieriem/can"
	inet "github.com/knieriem/can/drv/socketcan/internal/netlink"
)

var conn *inet.Conn

func main() {
	flag.Parse()

	var r io.Reader
	prompt := ""

	if flag.NArg() > 0 {
		r = strings.NewReader(strings.Join(flag.Args(), " "))
	} else {
		r = os.Stdin
		prompt = "% "
	}

	c, err := inet.Dial()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	conn = c

	if prompt != "" {
		fmt.Print(prompt)
	}
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			if prompt != "" {
				fmt.Print(prompt)
			}
			continue
		}
		f := strings.Fields(line)
		cmd, args := f[0], f[1:]
		cmd = strings.TrimSpace(cmd)
		if iLast := len(args) - 1; iLast >= 0 {
			args[iLast] = strings.TrimSpace(args[iLast])
		}
		if cmd == "exit" {
			break
		}
		err := handleCmd(cmd, args...)
		if err != nil {
			fmt.Printf("error: %v\n", err)
		}
		fmt.Print(prompt)
	}
}

var intfList = make(map[string]*inet.Interface)

var funcs = map[string]command{
	".conf": config,

	".up":   updown,
	".down": updown,

	"list": func(_ *inet.Interface, _ string, args ...string) error {
		list, err := inet.List()
		if err != nil {
			return err
		}
		for _, link := range list {
			fmt.Println(link.Name(), link.DriverName())
		}
		return nil
	},
}

type command func(intf *inet.Interface, name string, args ...string) error

func handleCmd(cmd string, args ...string) error {
	var intf *inet.Interface
	if strings.HasPrefix(cmd, ".") {
		// cmd is interface name
		intfName := cmd[1:]
		i, ok := intfList[intfName]
		if !ok {
			i1, err := conn.OpenInterface(intfName)
			if err != nil {
				return err
			}
			i = i1
		}
		intf = i
		if len(args) == 0 {
			cmd = "_default"
		} else {
			cmd, args = args[0], args[1:]
		}
	}
	fn, ok := funcs["."+cmd]
	if ok {
		if intf == nil {
			return errors.New("missing object")
		}
	} else {
		fn, ok = funcs[cmd]
		if !ok {
			return errors.New("command not found")
		}
	}
	return fn(intf, cmd, args...)
}

func config(intf *inet.Interface, _ string, args ...string) error {
	conf, err := can.ParseConfig(args...)
	if err != nil {
		return err
	}
	info, err := intf.Info()
	if err != nil {
		return err
	}
	ctl := info.Can.Controller()
	err = conf.ResolveBitTiming(ctl)
	if err != nil {
		return err
	}
	return intf.SetConfig(conf)
}

func updown(link *inet.Interface, name string, _ ...string) error {
	up := false
	if name == "up" {
		up = true
	}
	return link.UpDown(up)
}
