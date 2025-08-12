//go:build ignore

package linux

/*
#include <linux/can.h>
*/
import "C"

type CanFrame C.struct_can_frame
type CanfdFrame C.struct_canfd_frame
