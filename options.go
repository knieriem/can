package can

import (
	"errors"
	"strconv"
)

type (
	Bitrate int32
)

func parseOptions(optFields []string) (list []interface{}, err error) {
	for _, s := range optFields {
		if s == "" {
			err = errors.New("empty option")
			return
		}
		if b, ok := parseBitrate(s); ok {
			list = append(list, b)
		} else {
			list = append(list, s)
		}
	}
	return
}

func parseBitrate(s string) (b Bitrate, ok bool) {
	c := 1
	iLast := len(s) - 1
	switch s[iLast] {
	case 'k':
		c = 1000
		s = s[:iLast]
	case 'M':
		c = 1000000
		s = s[:iLast]
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return
	}
	b = Bitrate(i) * Bitrate(c)
	ok = true
	return
}
