// Convenience package that registers all known drivers.
package all

import (
	_ "can/drv/can4linux"
	_ "can/drv/canrpc"
	_ "can/drv/pcan"
	_ "can/drv/rnet"
)
