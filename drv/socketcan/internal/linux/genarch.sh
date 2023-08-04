set -e

pkg=linux
OS=$GOOS
ARCH=$GOARCH

if test $GOARCH = arm; then
	CC=arm-linux-gnueabi-gcc
	export CC
fi

# note: cgo execution depends on $GOARCH value
go tool cgo -godefs types.go  |
	sed '/^.. cgo -godefs/s,[^ ]\+/types.go,types.go,' |
	gofmt >ztypes_${OS}_$ARCH.go


(
	cat <<EOF
package $pkg
/*
#include <linux/can.h>
#include <linux/can/error.h>
*/
import "C"

const (
EOF
	<const awk '
		/^[^\/]/ { print "\t" $1 "= C." $1 ; next}
		{ print }
	'
	echo ')'
) > ,,const.go

go tool cgo -godefs ,,const.go |
	sed '/^.. cgo -godefs/s/[^ ]\+const.go/,,const.go/' |
	gofmt > zconst_${OS}_$ARCH.go
rm -f ,,const.go
