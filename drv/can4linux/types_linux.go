// Copyright 2012 The can Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package can4linux

/*
typedef unsigned char uchar;
typedef unsigned long ulong;
typedef unsigned int uint;

#include <sys/time.h>
typedef
struct Msg {
	int	flags;
	int	_;
	ulong id;
	struct timeval  tstamp;
	short	length;
	uchar data[8];
} Msg;

typedef
struct IoctlCmd {
	int cmd;
	int error;
	ulong ret;
} IoctlCmd;

typedef
struct IoctlConf {
	int name;
	ulong arg1;
	long arg2;
	int err;
	ulong ret;
} IoctlConf;

struct BufStatus {
	uint	size;
	uint	used;
};
long c;
typedef
struct IoctlStatus {
	uint	baud;
	uint	statusReg;
	uint	errWarningLimit;
	uint	numRxErrors;
	uint	numTxErrors;
	uint	errCodeReg;
	struct BufStatus rxBuf, txBuf;

	ulong ret;
	uint	controllerType;
} IoctlStatus;

*/
import "C"

type timeval C.struct_timeval
type msg C.Msg

type ioctlCmdArg C.IoctlCmd
type ioctlConfArg C.IoctlConf
type ioctlStatusArg C.IoctlStatus
type bufStatus C.struct_BufStatus
