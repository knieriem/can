# Copyright 2012 The can Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

pkg=api
OS=$GOOS

mksyscall=`go list -f '{{.Dir}}' syscall`/mksyscall.pl

case $GOARCH in
386)
	arch=-l32
	;;
amd64)
	arch=
	;;
*)
	echo GOARCH $GOARCH not supported
	exit 1
	;;
esac

GCC=

SFX=_${OS}_$GOARCH.go

perl $mksyscall $arch -tags $OS,$GOARCH ${pkg}_$OS.go |
	sed '/^import/a \
		import "syscall"' |
	sed 's/Syscall/syscall.Syscall/' |
	sed 's/SYS_/syscall.SYS_/' |
	sed '/import *"unsafe"/d' |
	sed '/package syscall/s,syscall,'$pkg, |
	sed '/^\/\/go:build/d' |
	gofmt > z$pkg$SFX

if test -f ,,lintypes.go; then
	# note: cgo execution depends on $GOARCH value
	GCC=$GCC go tool cgo -godefs ,,lintypes.go |
		sed '/cgo.-godefs/s,'`pwd`/,, |
		sed '/VersionString/s,\]int8,]uint8,' |
		gofmt >ztypes$SFX
fi
