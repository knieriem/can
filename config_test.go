package can

import (
	"reflect"
	"testing"
	"time"

	"github.com/knieriem/can/timing"
)

// Helper function to create an Optional[bool] with a value.
func newOptionalBool(value bool) Optional[bool] {
	return Optional[bool]{Valid: true, Value: value}
}

// Helper function to create a BitTimingConfig for tests.
func newBitTimingConfig(bitrate uint32, sp timing.SamplePoint, tq time.Duration, prescaler, prop, ps1, ps2, sjw int) BitTimingConfig {
	return BitTimingConfig{
		Bitrate:     bitrate,
		SamplePoint: sp,
		BitTiming: timing.BitTiming{
			Prescaler: prescaler,
			PropSeg:   prop,
			PhaseSeg1: ps1,
			PhaseSeg2: ps2,
			SJW:       sjw,
		},
		Tq: tq,
	}
}

// deepEqualConfig compares two Config structs for equality.
// It is used instead of reflect.DeepEqual to handle cases where
// optional fields might be zero-valued but invalid.
func deepEqualConfig(a, b *Config) bool {
	// Nominal BitTimingConfig comparison
	if a.Nominal.Bitrate != b.Nominal.Bitrate ||
		a.Nominal.SamplePoint != b.Nominal.SamplePoint ||
		a.Nominal.Tq != b.Nominal.Tq ||
		!reflect.DeepEqual(a.Nominal.BitTiming, b.Nominal.BitTiming) {
		return false
	}

	// Data BitTimingConfig comparison
	if a.Data.Valid != b.Data.Valid {
		return false
	}
	if a.Data.Valid {
		if a.Data.Value.Bitrate != b.Data.Value.Bitrate ||
			a.Data.Value.SamplePoint != b.Data.Value.SamplePoint ||
			a.Data.Value.Tq != b.Data.Value.Tq ||
			!reflect.DeepEqual(a.Data.Value.BitTiming, b.Data.Value.BitTiming) {
			return false
		}
	}

	// Optional[bool] fields
	if a.Termination.Valid != b.Termination.Valid ||
		(a.Termination.Valid && a.Termination.Value != b.Termination.Value) {
		return false
	}
	if a.FDMode.Valid != b.FDMode.Valid ||
		(a.FDMode.Valid && a.FDMode.Value != b.FDMode.Value) {
		return false
	}

	return true
}

func TestParseConfig(t *testing.T) {
	// Test cases for a variety of valid configuration strings
	tests := []struct {
		name    string
		input   string
		want    *Config
		wantFmt string // The expected string after formatting
		wantErr bool
	}{
		{
			name:  "nominal 500k",
			input: "500k",
			want: &Config{
				Nominal: newBitTimingConfig(500000, 0, 0, 0, 0, 0, 0, 0),
			},
			wantFmt: "500k",
		},
		{
			name:  "nominal 1M with sample point",
			input: "1M@.875",
			want: &Config{
				Nominal: newBitTimingConfig(1000000, 875, 0, 0, 0, 0, 0, 0),
			},
			wantFmt: "1M@.875",
		},
		{
			name:  "nominal 250k with SJW",
			input: "b:250k:s4",
			want: &Config{
				Nominal: newBitTimingConfig(250000, 0, 0, 0, 0, 0, 0, 4),
			},
			wantFmt: "250k:s4",
		},
		{
			name:  "nominal with full bit timing",
			input: "b:/20:8-7-6",
			want: &Config{
				Nominal: newBitTimingConfig(0, 0, 0, 20, 8, 7, 6, 0),
			},
			wantFmt: "/20:8-7-6",
		},
		{
			name:  "data 2M with sample point and SJW",
			input: "db:2M@.75:s4 b:500k",
			want: &Config{
				Nominal: newBitTimingConfig(500000, 0, 0, 0, 0, 0, 0, 0),
				Data:    Optional[BitTimingConfig]{Valid: true, Value: newBitTimingConfig(2000000, 750, 0, 0, 0, 0, 0, 4)},
				FDMode:  newOptionalBool(true),
			},
			wantFmt: "500k db:2M@.75:s4",
		},
		{
			name:    "data bitrate without nominal bitrate should fail",
			input:   "db:2M@.75",
			want:    nil,
			wantErr: true,
		},
		{
			name:  "simple fd",
			input: "2000k fd",
			want: &Config{
				Nominal: newBitTimingConfig(2e6, 0, 0, 0, 0, 0, 0, 0),
				FDMode:  newOptionalBool(true),
			},
			wantFmt: "2M fd",
		},
		{
			name:  "fd disabled",
			input: "250k fd:0",
			want: &Config{
				Nominal: newBitTimingConfig(250e3, 0, 0, 0, 0, 0, 0, 0),
				FDMode:  newOptionalBool(false),
			},
			wantFmt: "250k fd:0",
		},
		{
			name:  "termination enabled",
			input: "500k T",
			want: &Config{
				Nominal:     newBitTimingConfig(500e3, 0, 0, 0, 0, 0, 0, 0),
				Termination: newOptionalBool(true),
			},
			wantFmt: "500k T",
		},
		{
			name:  "termination disabled",
			input: "250k T0",
			want: &Config{
				Nominal:     newBitTimingConfig(250e3, 0, 0, 0, 0, 0, 0, 0),
				Termination: newOptionalBool(false),
			},
			wantFmt: "250k T0",
		},
		{
			name:  "complex config string",
			input: "b500k fd T db2M@.75:s4",
			want: &Config{
				Nominal:     newBitTimingConfig(500e3, 0, 0, 0, 0, 0, 0, 0),
				Data:        Optional[BitTimingConfig]{Valid: true, Value: newBitTimingConfig(2e6, 750, 0, 0, 0, 0, 0, 4)},
				Termination: newOptionalBool(true),
				FDMode:      newOptionalBool(true),
			},
			wantFmt: "500k db:2M@.75:s4 T",
		},
		{
			name:    "invalid key",
			input:   "invalid",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid boolean value",
			input:   "fd:2",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConfig(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return // Don't check for equality if an error is expected
			}

			// Check if the parsed Config struct matches the expected Config struct
			if !deepEqualConfig(got, tt.want) {
				t.Errorf("ParseConfig() got = %+v, want %+v", got, tt.want)
			}

			// Check for round-trip: convert the parsed Config back to a string
			gotFmt := got.Format(" ")
			if gotFmt != tt.wantFmt {
				t.Errorf("Format() got = %q, want %q", gotFmt, tt.wantFmt)
			}
		})
	}
}
