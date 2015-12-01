package can

import (
	"strconv"
)

type (
	Bitrate     int32
	Termination bool
)

func parseOptions(optFields []string) (list []interface{}, err error) {
	for _, s := range optFields {
		if s == "" {
			err = Error("empty option")
			return
		}
		var opt interface{}
		if b, ok := parseBitrate(s); ok {
			opt = b
		} else if s == "T" {
			opt = Termination(true)
		} else {
			opt = s
		}
		list = append(list, opt)
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
