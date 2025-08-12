package can

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/knieriem/can/timing"
)

type Config struct {
	Nominal BitTimingConfig
	Data    Optional[BitTimingConfig]

	Termination Optional[bool]
	FDMode      Optional[bool]
}

type BitTimingConfig struct {
	Bitrate     uint32
	SamplePoint timing.SamplePoint

	timing.BitTiming

	Tq time.Duration
}

type Optional[T any] struct {
	Valid bool
	Value T
}

func (o *Optional[T]) Set(value T) {
	o.Valid = true
	o.Value = value
}

func (bt *BitTimingConfig) isUnset() bool {
	return bt.Bitrate == 0 && bt.BitTiming.PhaseSeg1 == 0
}

// ParseConfSpecs parses CAN adapter configuration specifications.
// The strings may contain space separated parameter settings.
//
// The syntax of configuration strings has been designed with Plan 9's _ctl_
// file commands in mind (see https://plan9.io/magic/man2html/3/uart
// for an example). The basic structure is:
//
//	key ":" value
//
// The colon may be omitted; in this case, the start of the value
// will be the position of the first decimal digit.
//
// A value may be a plain integer, a bool or a more complex string.
//
// Boolean values are represented by integer values 1 and 0,
// which map to true and false. In case of true, the value may be omitted --
// the key used alone stands for the value being "true".
//
// If a parameter is omitted altoget*her, the default settings of adapters
// will be used, if not otherwise specified.
//
// Defined parameters:
//
//	b - nominal bit timing, optionally with a sample point, and SJW
//
//		A value can be a bit timing expression:
//
//		  bit-timing-expr = ( bitrate | bittiming ) [[ ":" ] sjw]
//
//		          bitrate = number [ "k" | "M" ] [ "@" sample-point ]
//
//		        bittiming = ( "*" tq | "/" prescaler ) ":" seg-expr
//
//		         seg-expr = prop-seg "-" ps1 "-" ps2
//
//		              sjw = "s" number
//
//		     sample-point = "." fraction
//
//		Examples: 500k@.875, b1M@.75 refering to 500 kbit/s or 1 Mbit/s,
//		with sample points at 87.5% resp. 75%.
//		In case of nominal bitrates, as an exception, the "b" key may be
//		omitted. So, stating "500k" will be recognized as b:500k.
//
//	db - data bit timing, optionally with a sample point, and SJW
//
//		A data bit timing value is a bit timing expr. See the definition
//		of "b" (nominal bit timing) for details.
//
//	fd - CAN FD mode
//
//		A boolean parameter deciding whether the CAN adapter should be run
//		in CAN 2.0 mode or FD mode.
//
//	T - enable/disable termination resistor
//
//		This is a boolean parameter.
func ParseConfig(specs ...string) (*Config, error) {
	var c Config

	any := false
	for _, s := range specs {
		s := strings.TrimSpace(s)
		if s == "" {
			continue
		}
		f := strings.Fields(s)
		for _, cs := range f {
			err := c.fromSpec(cs)
			if err != nil {
				return nil, err
			}
		}
		any = true
	}

	if !any {
		return nil, nil
	}

	if c.Data.Valid {
		if !c.FDMode.Valid {
			c.FDMode.Set(true)
		}
	}
	if c.Nominal.isUnset() {
		return nil, errors.New("missing nominal bitrate")
	}

	return &c, nil
}

func (c *Config) fromSpec(s string) error {

	b := s[0]
	if b >= '1' && b <= '9' || b == '*' || b == '/' {
		return c.Nominal.fromString(s)
	}

	iColon := indexColon(s)
	key, value := s, "1"
	if iColon == -1 {
		if iInt := strings.IndexAny(s, "0123456789"); iInt != -1 {
			key, value = s[:iInt], s[iInt:]
		} else if !slices.Contains(boolKeys, s) {
			return fmt.Errorf("key not found: %q", s)
		}
	} else {
		key, value = s[:iColon], s[iColon+1:]
	}
	switch key {
	case "b":
		err := c.Nominal.fromString(value)
		if err != nil {
			return err
		}
	case "db":
		err := c.Data.Value.fromString(value)
		if err != nil {
			return err
		}
		c.Data.Valid = true
	case "fd":
		err := parseBoolInt(&c.FDMode, value)
		if err != nil {
			return err
		}
	case "T":
		err := parseBoolInt(&c.Termination, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func indexColon(s string) int {
	iColon := strings.IndexByte(s, ':')
	if iColon == -1 {
		return -1
	}
	if iAt := strings.IndexByte(s, '@'); iAt != -1 {
		if iAt < iColon {
			return -1
		}
	}
	return iColon
}

var boolKeys = []string{"fd", "T"}

func parseBoolInt(dest *Optional[bool], s string) error {
	if s == "1" {
		dest.Valid = true
		dest.Value = true
		return nil
	}
	if s == "0" {
		dest.Valid = true
		dest.Value = false
		return nil
	}
	return fmt.Errorf("cannot parse %q as boolean value", s)
}

type tqSpec struct {
	name string
	key  byte
	next byte
	arg  *int
}

func (c *BitTimingConfig) fromString(s string) error {
	d := stringDecoder{s: s}

	b := s[0]
	var tq int
	tqSpecs := map[byte]*tqSpec{
		'*': {name: "tq", key: '*', next: '/', arg: &tq},
		'/': {name: "prescaler", key: '/', next: '*', arg: &c.BitTiming.Prescaler},
	}
	tqSpec, ok := tqSpecs[b]
	if !ok {
		v, err := d.parsePrefixedInt(0, "bitrate", false)
		if err != nil {
			return err
		}
		bitrate := uint32(v)
		s := d.s
		if len(s) != 0 {
			switch s[0] {
			case 'M':
				bitrate *= 1e6
				d.s = s[1:]
			case 'k':
				bitrate *= 1000
				d.s = s[1:]
			}
			s = d.s
		}
		c.Bitrate = bitrate
		if len(s) > 0 {
			if s[0] == '@' {
				d.s = s[1:]
				sp, err := d.parsePrefixedInt('.', "samplePoint", false)
				if err != nil {
					return err
				}
				c.SamplePoint = timing.SamplePoint(sp)
			}
		}
	} else {
		err := d.parseTqArg(tqSpec, false)
		if err != nil {
			return err
		}

		if s := d.s; len(s) > 0 {
			b := s[0]
			if b == tqSpec.next {
				tqSpec, ok := tqSpecs[tqSpec.next]
				if ok {
					err := d.parseTqArg(tqSpec, true)
					if err != nil {
						return err
					}
				}
			}
		}

		c.PropSeg, err = d.parsePrefixedInt(':', "prop", false)
		if err != nil {
			return err
		}
		c.PhaseSeg1, err = d.parsePrefixedInt('-', "ps1", false)
		if err != nil {
			return err
		}
		c.PhaseSeg2, err = d.parsePrefixedInt('-', "ps2", false)
		if err != nil {
			return err
		}
	}

	if tq != 0 {
		c.Tq = time.Duration(tq) * time.Nanosecond
	}

	if strings.HasPrefix(d.s, ":s") {
		d.s = d.s[1:]
	}
	u, err := d.parsePrefixedInt('s', "sjw", true)
	if err != nil {
		return err
	}
	c.SJW = u

	return nil
}

type stringDecoder struct {
	s string
}

func (d *stringDecoder) parseTqArg(s *tqSpec, optional bool) error {
	v, err := d.parsePrefixedInt(s.key, s.name, optional)
	if err != nil {
		return err
	}
	*s.arg = v
	return nil
}

func (d *stringDecoder) parsePrefixedInt(prefix byte, key string, optional bool) (int, error) {
	if len(d.s) < 2 {
		if optional {
			return 0, nil
		}
		return 0, fmt.Errorf("parsing %q: missing parameter", key)
	}
	s := d.s
	if prefix != 0 {
		if s[0] != prefix {
			if optional {
				return 0, nil
			}
			return 0, fmt.Errorf("parsing %q: missing prefix %q", key, prefix)
		}
		s = s[1:]
	}

	digit0 := byte('1')
	if prefix == '.' {
		digit0 = '0'
	}
	if b := s[0]; b < digit0 || b > '9' {
		return 0, fmt.Errorf("parsing %q: missing number", key)
	}
	i := 0
	for _, b := range s[1:] {
		if b < '0' || b > '9' {
			break
		}
		i++
	}
	s, d.s = s[:1+i], s[1+i:]
	u, err := strconv.ParseUint(s, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("parsing %q: %w", key, err)
	}
	if prefix == '.' {
		diff := len(s) - 3
		if diff >= 0 {
			for range diff {
				u = (u + 5) / 10
			}
		} else {
			for range -diff {
				u *= 10
			}
		}
	}
	return int(u), err
}

func (c *Config) Format(sep string) string {
	skipFD := false
	enc := configEncoder{}

	enc.addValue("b", c.Nominal.String())
	if c.Data.Valid {
		enc.addValue("db", c.Data.Value.String())
		if c.FDMode.Valid && c.FDMode.Value {
			skipFD = true
		}
	}

	if !skipFD {
		enc.addOptBool("fd", c.FDMode)
	}
	enc.addOptBool("T", c.Termination)
	return strings.Join(enc.buf, sep)
}

func (c *BitTimingConfig) String() string {
	sjw := ""
	if c.SJW != 0 {
		sjw += ":s" + strconv.Itoa(c.SJW)
	}
	if b := c.Bitrate; b != 0 {
		sp := ""
		if c.SamplePoint != 0 {
			sp = "@." + fmt.Sprintf("%03d", c.SamplePoint)
			sp = strings.TrimRight(sp, "0")
		}
		return formatBitrate(b) + sp + sjw
	}

	x := ""
	if c.Tq != 0 {
		x = "*" + strconv.FormatInt(c.Tq.Nanoseconds(), 10) + ":"
	} else if c.Prescaler > 1 {
		x = "/" + strconv.FormatInt(int64(c.Prescaler), 10) + ":"
	}
	x += strconv.Itoa(c.PropSeg) + "-" + strconv.Itoa(c.PhaseSeg1) + "-" + strconv.Itoa(c.PhaseSeg2)
	return x + sjw
}

func formatBitrate(b uint32) string {
	suffix := ""
	if b%1e6 == 0 {
		suffix = "M"
		b /= 1e6
	} else if b%1e3 == 0 {
		suffix = "k"
		b /= 1e3
	}
	return strconv.FormatUint(uint64(b), 10) + suffix
}

type configEncoder struct {
	sb  strings.Builder
	buf []string
}

func (enc *configEncoder) addValue(key string, value string) {
	pfx := ""
	if key != "b" {
		pfx = key + ":"
	}
	enc.buf = append(enc.buf, pfx+value)
}

func (enc *configEncoder) addOptBool(key string, opt Optional[bool]) {
	if !opt.Valid {
		return
	}
	s := key
	if !opt.Value {
		if len(key) != 1 {
			s += ":0"
		} else {
			s += "0"
		}
	}
	enc.buf = append(enc.buf, s)
}
