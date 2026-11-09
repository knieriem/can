package can

import (
	"slices"
	"testing"
)

type namedDriver struct {
	unsupported
	name string
}

func (d namedDriver) Name() string { return d.name }

// TestRegisterDriverReplaces protects the rule that lets a program
// import drv/all and then register a driver with its own options:
// a driver with the name of an earlier one replaces it.
func TestRegisterDriverReplaces(t *testing.T) {
	saved := drvlist
	defer func() { drvlist = saved }()
	drvlist = nil

	a, b, a2 := namedDriver{name: "a"}, namedDriver{name: "b"}, namedDriver{name: "a"}
	RegisterDriver(a)
	RegisterDriver(b)
	RegisterDriver(a2)
	if want := []Driver{a2, b}; !slices.Equal(drvlist, want) {
		t.Errorf("drvlist = %v, want %v", drvlist, want)
	}
}
