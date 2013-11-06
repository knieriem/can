# Copyright 2012 The can Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

pkg=can4linux
OS=$GOOS
ARCH=$GOARCH

mksyscall=`go list -f '{{.Dir}}' syscall`/mksyscall.pl

perl $mksyscall ${pkg}_$OS.go |
	sed 's/^package.*syscall$/package can4linux/' |
	sed '/^import/a \
		import "syscall" ' |
	sed 's/Syscall/syscall.Syscall/' |
	sed 's/SYS_/syscall.SYS_/' |
	gofmt > z${pkg}_${OS}_$ARCH.go

# note: cgo execution depends on $GOARCH value
go tool cgo -godefs types_$OS.go  |
	gofmt >ztypes_${OS}_$ARCH.go

