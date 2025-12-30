//go:build linux
package socketcan

import (
	"io"
	"os"
	"os/exec"

	"github.com/knieriem/can"
	"github.com/knieriem/can/drv/socketcan/internal/linerpc"
	"github.com/knieriem/can/drv/socketcan/internal/netlink"
)

type privilegedAccess interface {
	UpDown(bool) error
	SetConfig(*can.Config) error
	Close() error
}

type privilegedDirect struct {
	*netlink.Interface
}

func (priv *privilegedDirect) Close() error {
	return nil
}

type privilegedUtil struct {
	cl       *linerpc.Client
	closer   io.Closer
	cmd      *exec.Cmd
	intfName string
}

func startPrivilegedUtil(name, intfName string) (*privilegedUtil, error) {
	cmd := exec.Command(name)
	w, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	r, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	cl := linerpc.NewClient(&readWriter{Reader: r, Writer: w})
	util := new(privilegedUtil)
	util.cl = cl
	util.closer = w
	util.cmd = cmd
	util.intfName = "." + intfName
	return util, nil
}

type readWriter struct {
	io.Reader
	io.Writer
}

func (rw *readWriter) Read(data []byte) (n int, err error) {
	n, err = rw.Reader.Read(data)
	return n, err
}

func (util *privilegedUtil) UpDown(up bool) error {
	if up {
		_, err := util.cl.Call(util.intfName, "up")
		return err
	}
	_, err := util.cl.Call(util.intfName, "down")
	return err
}

func (util *privilegedUtil) SetConfig(c *can.Config) error {
	_, err := util.cl.Call(util.intfName, "conf", c.Format(" "))
	return err
}

func (util *privilegedUtil) Close() error {
	err := util.closer.Close()
	err1 := util.cmd.Wait()
	if err1 != nil {
		return err1
	}
	return err
}
