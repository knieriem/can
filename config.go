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

	MsgFilter []MsgFilter
}

type BitTimingConfig struct {
	Bitrate     uint32
	SamplePoint timing.SamplePoint

	timing.BitTiming

	Tq time.Duration
}

func formatSJW(c *timing.BitTiming) string {
	if c.SJW != 0 {
		if c.SJW == c.PhaseSeg2 {
			return ""
		}
	}
	if c.SJWExt.Ratio1000 != 0 {
		s := ":s." + fmt.Sprintf("%03d", c.SJWExt.Ratio1000)
		return strings.TrimRight(s, "0")
	}
	if c.SJW != 0 {
		return ":s" + strconv.Itoa(c.SJW)
	}
	return ""
}

type Optional[T any] struct {
	Valid bool
	Soft  bool
	Value T
}

func (o *Optional[T]) Set(value T) {
	o.Valid = true
	o.Value = value
}

func (bt *BitTimingConfig) isUnset() bool {
	return bt.Bitrate == 0 && bt.BitTiming.PhaseSeg1 == 0
}

// clone creates a copy of the Config value;
// not that the message filters slice is not duplicated currently.
func (conf *Config) clone() *Config {
	cp := *conf
	return &cp
}

// ResolveFDMode determines whether a Config requests FD mode,
// factoring in the hardware's FD capability. It does not modify the Config.
//
// FD mode is requested if either the FDMode option is enabled or data bit timing
// is specified. If the request is "soft" (FDMode.Soft or Data.Soft is true)
// and fdCapable is false, the function returns isFD==false without an error.
// If the request is strict and fdCapable is false, it returns ErrFDNotSupported.
func (conf *Config) ResolveFDMode(fdCapable bool) (isFD bool, err error) {
	if conf.FDMode.Valid && conf.FDMode.Value {
		if !fdCapable {
			if conf.FDMode.Soft {
				return false, nil
			}
			return false, ErrFDNotSupported
		}
		return true, nil
	}
	if conf.Data.Valid {
		if !fdCapable {
			if conf.Data.Soft {
				return false, nil
			}
			return false, ErrFDNotSupported
		}
		return true, nil
	}
	return false, nil
}

var ErrFDNotSupported = Error("FD mode not supported")

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
//		              sjw = "s" [ number | "." fraction ]
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
//	f - CAN message filter
//
//		A value has the form:  id ":" mask,
//		where id and mask either consist of three characters (standard
//		frame) or up to eight characters (extended frame), as in:
//			f:123:7ff or f:123_4567:1fff_ffff
//
//		A short form can be used where only the id is specified and
//		":" mask part is omitted. Within the id part, "-" may be used
//		for a nibble that may contain any value from 0 to 0xF, as in:
//			f:67-
//		This would enable the receipt of messages with standard frame
//		CAN IDs from 0x670 to 0x67F.
//
//		Multiple filters may be specified. The effect of a filter may
//		be inverted by using a prefix "!" in front of the id part,
//		like in "f!12-" (the example would avoid the reception of
//		standard frames in the range 120 to 12F).
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
			c.FDMode.Soft = c.Data.Soft
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

	soft := false
	allowSoft := false
	iColon := indexColon(s)
	key, value := s, "1"
	if iColon == -1 {
		if iInt := strings.IndexAny(s, "!0123456789"); iInt != -1 {
			key, value = s[:iInt], s[iInt:]
		} else {
			key, soft = strings.CutSuffix(key, "?")

			if !slices.Contains(boolKeys, key) {
				return fmt.Errorf("key not found: %q", key)
			}
		}
	} else {
		key, value = s[:iColon], s[iColon+1:]
	}
	if !soft {
		key, soft = strings.CutSuffix(key, "?")
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
		c.Data.Soft = soft
		allowSoft = true
	case "fd":
		err := parseBoolInt(&c.FDMode, value)
		if err != nil {
			return err
		}
		c.FDMode.Soft = soft
		allowSoft = true
	case "f":
		f, err := parseMsgFilter(value)
		if err != nil {
			return err
		}
		c.MsgFilter = append(c.MsgFilter, *f)
	case "T":
		err := parseBoolInt(&c.Termination, value)
		if err != nil {
			return err
		}
		c.Termination.Soft = soft
		allowSoft = true
	}
	if soft && !allowSoft {
		return fmt.Errorf("key %q may not be used with '?'", key)
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

	// parse sync jump width
	if strings.HasPrefix(d.s, ":s") {
		d.s = d.s[1:]
	}
	if strings.HasPrefix(d.s, "s.") {
		d.s = d.s[1:]
		rDiv1000, err := d.parsePrefixedInt('.', "sjw", false)
		if err != nil {
			return err
		}
		c.SJW = 0
		c.SJWExt.Ratio1000 = rDiv1000
	} else {
		u, err := d.parsePrefixedInt('s', "sjw", true)
		if err != nil {
			return err
		}
		c.SJW = u
		c.SJWExt = timing.SJWExt{}
	}

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
	sjw := formatSJW(&c.BitTiming)
	if b := c.Bitrate; b != 0 {
		sp := ""
		if c.SamplePoint != 0 {
			sp = "@." + fmt.Sprintf("%03d", c.SamplePoint)
			sp = strings.TrimRight(sp, "0")
		}
		return FormatBitrate(b) + sp + sjw
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

func FormatBitrate(b uint32) string {
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

// ResolveBittiming calls Resolve on the nominal and, if requested and
// supported, the data [BitTimingConfig] fields, updating the
// Config in-place. The function returns any error received from any
// of the Resolve calls.
func (conf *Config) ResolveBitTiming(ctl *timing.Controller) error {
	wantFD := (conf.FDMode.Valid && conf.FDMode.Value) || conf.Data.Valid
	haveFD := ctl.Data != nil
	if wantFD && !haveFD {
		wantFD = false
	}
	if wantFD {
		if !conf.Data.Valid {
			conf.Data.Value = conf.Nominal
			conf.Data.Valid = true
		}
	}
	err := conf.Nominal.Resolve(nil, ctl.Clock, &ctl.Nominal)
	if err != nil {
		return err
	}
	if wantFD {
		err := conf.Data.Value.Resolve(nil, ctl.Clock, ctl.Data)
		if err != nil {
			return err
		}
	}
	return nil
}

// Resolve interprets a BitTimingConfig.
//
// If a bitrate and (optionally) a sample point are specified,
// it calculates a [timing.BitTiming], taking fOsc and dev into account.
//
// If, instead, the BitTiming field is provided, it validates and, if
// necessary, fills in the Tq or Prescaler fields.
//
// The result is stored into dest if dest is non-nil; otherwise,
// the receiver btc is modified in-place.
func (btc *BitTimingConfig) Resolve(dest *BitTimingConfig, clock uint32, cstr *timing.Constraints) error {
	if dest == nil {
		dest = btc
	}
	if btc.Tq != 0 {
		ps := int((btc.Tq*time.Duration(clock) + time.Second/2 - 1) / time.Second)
		if dest != btc {
			*dest = *btc
		}
		if btc.Prescaler != 0 {
			if ps != btc.Prescaler {
				return errors.New("prescaler mismatch")
			}
		} else {
			dest.Prescaler = int(ps)
		}
		return nil
	}
	if btc.PropSeg != 0 {
		// Tq is zero
		num := time.Second * time.Duration(btc.Prescaler)
		if dest != btc {
			*dest = *btc
		}
		dest.Tq = (num + time.Duration(clock)/2 - 1) / time.Duration(clock)
		return nil
	}
	if btc.Bitrate == 0 {
		return errors.New("bitrate not found")
	}
	t, err := timing.CalcBitTiming(clock, btc.Bitrate, btc.SamplePoint, cstr, timing.PreferLowerPrescaler())
	if err != nil {
		return err
	}
	t.SJW = btc.SJW
	t.SJWExt = btc.SJWExt
	t.ConstrainSJW(cstr.SJWMax)
	dest.BitTiming = *t
	dest.Tq = t.CalcTq(clock)
	return nil
}

type MsgFilter struct {
	ID       uint32
	IDMask   uint32
	ExtFrame bool
	Invert   bool
}

func parseMsgFilter(v string) (*MsgFilter, error) {
	var id, mask uint32
	extFrame := false

	v, invert := strings.CutPrefix(v, "!")
	i := strings.IndexByte(v, ':')
	if i != -1 {
		n := len(v) - strings.Count(v, "_")
		if n > 6+1 {
			extFrame = true
		}
		u, err := strconv.ParseUint("0x"+v[:i], 0, 32)
		if err != nil {
			return nil, err
		}
		id = uint32(u)
		u, err = strconv.ParseUint("0x"+v[i+1:], 0, 32)
		if err != nil {
			return nil, err
		}
		mask = uint32(u)
	} else {
		n := len(v)
		nDigit := 0
		pos := 0
		for i := range n {
			if pos > 28 {
				return nil, errors.New("id range exceeded")
			}
			b := v[n-1-i:]
			b = b[:1]
			switch b[0] {
			case '_':
				if i == 4 {
					continue
				}
				return nil, errors.New("syntax error")
			case '-':
				nDigit++
				pos += 4
				continue
			}
			u, err := strconv.ParseUint(b, 16, 8)
			if err != nil {
				return nil, err
			}
			nDigit++
			id |= uint32(u) << pos
			maskNibble := uint32(0xF)
			if pos == 28 {
				maskNibble = 1
			}
			mask |= maskNibble << pos
			pos += 4
		}
		if nDigit > 3 {
			extFrame = true
		}
	}
	// mask must contain id
	if mask|id != mask {
		return nil, errors.New("id not covered by mask")
	}

	return &MsgFilter{ID: id, IDMask: mask, ExtFrame: extFrame, Invert: invert}, nil
}

// Range returns two values defining a range that corresponds
// a single region defined by ID and IDMask fields. In this
// case ok will be set to true. If ID and IDMask would create
// multiple / many regions, ok is set to false.
// Range may be useful for drivers that implement filtering
// based on ID ranges.
func (f *MsgFilter) Range() (from, to uint32, ok bool) {
	topMask := uint32(1 << (11 - 1))
	if f.ExtFrame {
		topMask = 1 << (29 - 1)
	}
	mask := f.IDMask
	upperMask := topMask | (topMask - 1)
	topBit := mask & topMask
	if !isTopDownMask(mask, upperMask, topBit) {
		return 0, 0, false
	}

	inverted := ^f.IDMask & upperMask
	return f.ID, f.ID | inverted, true
}

func isTopDownMask(mask, upperMask, topBit uint32) bool {
	if mask == upperMask {
		return true
	}
	if mask == 0 {
		return true
	}

	// Flip bits: 11110000 -> 00001111
	inverted := ^mask & upperMask

	// Check if inverted is a block of 1s at the bottom (e.g., 00001111)
	// A block of 1s at the bottom + 1 is always a power of 2 (e.g., 00010000)
	return (inverted+1)&inverted == 0 && topBit != 0
}
