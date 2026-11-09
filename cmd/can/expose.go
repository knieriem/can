// Copyright 2013 The hgo Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/url"
	"strings"
	"time"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/canrpc"
	"github.com/knieriem/tool"

	"github.com/coder/websocket"
	"github.com/hashicorp/yamux"
)

var clientID string

var cmdExpose = &tool.Command{
	UsageLine: "expose [-id CLIENT_ID] URL [device]",
	Short:     "expose a CAN device via a yamux based tunnel",
	Long: `
Expose connects to a tunnel server at URL and serves the CAN device
to canrpc clients that the tunnel server forwards. When the tunnel is
lost, expose connects again, with a growing delay while connections
keep failing.
`,
	ExtraArgsReq: 1,
	ExtraArgsMax: 2,
}

func init() {
	cmdExpose.Run = runExpose
	cmdExpose.Flag.StringVar(&clientID, "id", "", "sent as X-Client-ID")
}

func runExpose(cmd *tool.Command, w io.Writer, args []string) error {
	devName := ""
	if len(args) > 1 {
		devName = args[1]
	}
	d, err := can.Open(devName, can.WithBusErrors()) // clients filter themselves
	if err != nil {
		return err
	}
	if err := canrpc.Register(rpc.DefaultServer, d, ""); err != nil {
		return err
	}

	// Keep the device reachable: when the tunnel is lost, dial again.
	// The delay grows while sessions keep failing quickly.
	const minDelay, maxDelay = time.Second, time.Minute
	delay := minDelay
	for {
		session, err := dialTunnel(args[0], clientID)
		if err != nil {
			log.Printf("expose: %v; retrying in %v", err, delay)
		} else {
			log.Print("expose: tunnel connected")
			start := time.Now()
			rpc.Accept(session) // returns when the session ends
			session.Close()
			if time.Since(start) > maxDelay {
				delay = minDelay
			}
			log.Printf("expose: tunnel lost; reconnecting in %v", delay)
		}
		time.Sleep(delay)
		delay = min(2*delay, maxDelay)
	}
}

func dialTunnel(rawURL, clientID string) (*yamux.Session, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	var conn net.Conn
	if strings.HasPrefix(u.Scheme, "http") {
		conn, err = dialHTTP(u, clientID)
	} else if strings.HasPrefix(u.Scheme, "ws") {
		conn, err = dialWebSocket(rawURL, clientID)
	} else {
		return nil, fmt.Errorf("unsupported url scheme %q", u.Scheme)
	}
	if err != nil {
		return nil, err
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to start yamux client: %w", err)
	}

	return session, nil
}

func dialHTTP(u *url.URL, clientID string) (net.Conn, error) {
	host := u.Host
	var conn net.Conn
	switch u.Scheme {
	case "http":
		if u.Port() == "" {
			host = host + ":80"
		}
		c, err := net.Dial("tcp", host)
		if err != nil {
			return nil, fmt.Errorf("dial failed: %w", err)
		}
		conn = c

	case "https":
		if u.Port() == "" {
			host = host + ":443"
		}
		c, err := tls.Dial("tcp", host, nil)
		if err != nil {
			return nil, fmt.Errorf("tls dial failed: %w", err)
		}
		conn = c

	default:
		return nil, fmt.Errorf("unsupported url scheme %q", u.Scheme)
	}

	req, err := http.NewRequest("GET", u.Path, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	req.Host = u.Host
	req.Header.Set("X-Client-ID", clientID)

	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, req)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("tunnel rejected by server: HTTP status %d", resp.StatusCode)
	}
	return &bufferedConn{Conn: conn, r: br}, nil
}

// bufferedConn reads through r, which may already hold the first
// tunnel bytes, received together with the HTTP response.
type bufferedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func dialWebSocket(tunnelURL, clientID string) (net.Conn, error) {
	ctx := context.Background()

	c, _, err := websocket.Dial(ctx, tunnelURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"X-Client-ID": []string{clientID}},
	})
	if err != nil {
		return nil, err
	}

	netConn := websocket.NetConn(ctx, c, websocket.MessageBinary)
	return netConn, nil
}
