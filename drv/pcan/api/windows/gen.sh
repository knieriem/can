# Copyright 2012 The can Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

pkg=api
OS=$GOOS

mksyscall=`go list -f '{{.Dir}}' syscall`/mksyscall_windows.pl

case $GOARCH in
386)
	gccarch=i686
	arch=-l32
	;;
amd64)
	gccarch=x86_64
	arch=
	;;
*)
	echo GOARCH $GOARCH not supported
	exit 1
	;;
esac

GCC=/usr/bin/$gccarch-w64-mingw32-gcc

SFX=_${OS}_$GOARCH.go

perl $mksyscall $arch ${pkg}_$OS.go |
	sed '/import *"DISABLEDunsafe"/d' |
	gofmt > z$pkg$SFX

if test -f windows/types.go; then
	# note: cgo execution depends on $GOARCH value
	GCC=$GCC go tool cgo -godefs windows/types.go  |
		gofmt >ztypes$SFX
fi
