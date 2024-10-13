#!/bin/sh

# Copyright 2012 The can Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

pkg=api

t=,,lintypes.go
cat <<EOF > $t
// +build ignore

package $pkg

/*
#include "pcan.h"
*/
import "C"

EOF

api() {
cat <<EOF
#Go type	.h type		Go name	.h ioctl name
Init		CANInit		_		INIT
Msg		CANMsg		WriteMsg	WRITE_MSG
RMsg	CANRdMsg	ReadMsg	READ_MSG
statusPar	STATUS		status		GET_STATUS
Diag		DIAG			_		_
Btr0Btr1	BTR0BTR1		SetBitrate	_
ExtStatus	EXTENDEDSTATUS	_	GET_EXT_STATUS
MsgFilter	MSGFILTER	SetMsgFilter	MSG_FILTER
Params	EXTRAPARAMS	SetExtraParams	EXTRA_PARAMS
EOF
}

>,,const

cat <<EOF >zmethods_linux.go
// Created by api-defs.sh - DO NOT EDIT
// $0

package $pkg

import (
	"unsafe"
)
EOF

api | while read gotype type method ioctl; do
	test $gotype = '#Go' && continue
	test $method = _ && method=$gotype
	test $ioctl = _ && ioctl=$type

	echo '	ioctl'$ioctl' = C.PCAN_'$ioctl >> ,,const
	echo type $gotype C.TP$type >> $t

	cat <<EOF >> zmethods_linux.go

func (f Fd) $method(p *$gotype) error {
	return ioctl(uintptr(f), ioctl$ioctl, uintptr(unsafe.Pointer(p)))
}
EOF

done

mtype() {
	uc=`echo $1 | tr a-z A-Z`
	test -z $2 || uc=$2
	echo '	'Msg$1' = C.MSGTYPE_'$uc
}

(
	echo 'const ('
	cat ,,const

	echo
	mtype Status
	mtype Extended
	mtype Rtr
	mtype Std STANDARD

	errors='XMTFULL
		OVERRUN
		BUSLIGHT
		BUSHEAVY
		BUSOFF
		QRCVEMPTY
		QOVERRUN
		QXMTFULL
		REGTEST
		NOVXD
		RESOURCE
		ILLPARAMTYPE
		ILLPARAMVAL
	'
	echo
	echo '	OK Status = C.CAN_ERR_OK'
	for e in $errors; do
		echo '	Err'$e' Status = C.CAN_ERR_'$e
	done

	echo ')'
) >> $t


rm -f ,,const
