package linerpc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

type Client struct {
	io.Writer

	mu sync.Mutex
	s  *bufio.Scanner
}

type Result []string

func NewClient(conn io.ReadWriter) *Client {
	cl := new(Client)
	s := bufio.NewScanner(conn)
	s.Split(splitAtPrompt)
	s.Scan()

	cl.Writer = conn
	cl.s = s
	return cl
}

func splitAtPrompt(data []byte, atEOF bool) (advance int, token []byte, err error) {
	i := bytes.Index(data, []byte("% "))
	if i == -1 {
		if atEOF {
			err = bufio.ErrFinalToken
		}
		return 0, nil, err
	}
	token = data[:i]

	advance = i + 2
	if advance == len(data) {
		if atEOF {
			err = bufio.ErrFinalToken
		}
	}
	return advance, token, err
}

func (cl *Client) Call(cmd string, args ...string) (Result, error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	_, err := fmt.Fprintln(cl, cmd, strings.Join(args, " "))
	if err != nil {
		return nil, err
	}
	if !cl.s.Scan() {
		return nil, cl.s.Err()
	}
	text := cl.s.Text()
	if len(text) == 0 {
		// success case without output
		return nil, nil
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line, found := strings.CutPrefix(line, "error: "); found {
			return nil, errors.New(line)
		}
	}
	return lines, nil
}
