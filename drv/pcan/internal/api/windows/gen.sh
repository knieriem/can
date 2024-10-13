# Copyright 2012 The can Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

set -e

pkg=api
OS=$GOOS
GOROOT=`go env GOROOT`

case $GOARCH in
386)
	gccarch=i686-w64-mingw32
	;;
amd64)
	gccarch=x86_64-w64-mingw32
	;;
*)
	echo GOARCH $GOARCH not supported
	exit 1
	;;
esac

GCC=/usr/bin/$gccarch-gcc

SFX=_${OS}_$GOARCH.go

src=${pkg}_$OS.go
mv $src _$src
sed '/^package/s,syscall,none,' <_$src >$src
GOARCH= GOOS= go run golang.org/x/sys/windows/mkwinsyscall $src |
	sed 's/^package.*none/package '$pkg'/' |
	gofmt > z$pkg$SFX
rm -f $src
mv _$src $src

#perl $mksyscall $arch ${pkg}_$OS.go |
#	sed '/import *"DISABLEDunsafe"/d' |
#	gofmt > z$pkg$SFX

if test -f windows/types.go; then
	# note: cgo execution depends on $GOARCH value
	CC=$GCC go tool cgo -godefs windows/types.go |
		sed '/cgo.-godefs/s,'`pwd`/,, |
		gofmt >ztypes$SFX
fi
