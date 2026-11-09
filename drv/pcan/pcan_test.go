package pcan

import (
	"errors"
	"testing"

	"github.com/knieriem/can"
)

// TestWrapErrUnwrap protects that errors.Is sees through pcan's
// wrapped errors. Without hardware, nothing else checks pcan errors.
func TestWrapErrUnwrap(t *testing.T) {
	err := error(can.ErrClosed)
	wrapErr("read", &err)
	if !errors.Is(err, can.ErrClosed) {
		t.Errorf("errors.Is(%v, ErrClosed) = false", err)
	}
	if got, want := err.Error(), "pcan: read: can: device closed"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
