# Copyright 2012 The can Authors.  All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

/Value def/ { proc = 1 }
/^.. PCAN devi/ {proc = 0 }
/^.. PCAN parameters/ { proc = 1; camel = 1 }
/Type def/ { exit }

/Currently defined and supported PCAN chan/ { type = "Handle" }
/PCAN error and stat/ { type = "Status"}
/PCAN devices/ {type = ""}
/Baud rate codes/ {type = ""; camel = 0}
$2 ~ /TYPE_/ { type = "HwType"}

/PCAN-.* interface, channel/ {sub("PCAN-", "", $5)}

proc && /#define/{
	sub("NONEBUS", "NoneBus")
	gsub("PARAMETER", "PARAM")
	gsub("PCAN_", "")
	gsub("ERROR_", "Err")
	gsub("MESSAGE_", "MSG_")
	gsub("STANDARD_", "STD_")
	gsub("EXTENDED_", "EXT_")
	sub("BAUD_", "Baud")
	sub("TYPE_", "Type")
	$1 = ""
	sub("LOG_FUNCTION_", "LOG_FN_", $2)
	ltype = ""
	if (type)
		ltype = type
	ltype = addtypes($2, ltype)
	sub("CHANNEL_", "CHAN_", $2)
	sub("5VOLTS", "FIVE_VOLTS", $2)
	if (camel) {
		$2 = camelize($2)
		if ($3 ~ /MSG_/)
			$3 = camelize($3)
	}
	if ($2 == "ErrOK")
		$2 = "OK"
	if (ltype)
		$2 = $2 " " ltype
	sub("$", " = ", $2)
	print
}

proc && /^ *$/ {
	print ""
}

function camelize(s, a) {
	split(s, a, "")
	for (i = 1; i <= length(a); i++) {
		if (i == 1) {
			s = a[i]
		} else {
			if (a[i] == "_") {
				s = s a[i+1]
				i++
			} else {
				s = s tolower(a[i])
			}
		}
	}
	return s
}
