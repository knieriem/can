// Convenience package that registers all known drivers.
package all

import (
	_ "github.com/knieriem/can/drv/canrpc"
	_ "github.com/knieriem/can/drv/pcan"
)
