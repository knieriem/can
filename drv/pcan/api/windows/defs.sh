# Copyright 2012 The can Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

W=./windows
m=param-map

fn=$1


header() {
cat <<EOF
// MACHINE GENERATED FROM "PCANBasic.h"; DO NOT EDIT
package api
const (
EOF
}

defs_windows() {
	header
	grep -v Baud
	echo ')'
}

defs() {
	header
	grep Baud
	echo ')'
}

awk -F'	' '
BEGIN {
	print "function addtypes(arg, t) {"
}
{
	print "if (arg == \"" $1 "\")"
	printf "	return \"%sPar\"\n", $2
}
END {
	print "return t\n}"
}' <$W/$m >,,$m.awk

awk -f $W/defs.awk -f ,,$m.awk < PCANBasic.h |
	def$fn |
	sed 's, *//.*$,,' |
	gofmt


rm -f ,,$m.awk
