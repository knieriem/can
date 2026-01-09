package canfd

import "github.com/knieriem/can"

func DLC(n int) (next byte, needsFD bool, err error) {
	if n <= 8 {
		return byte(n), false, nil
	}

	needsFD = true
	for i, nFD := range can.ValidFDSizes {
		if n == nFD {
			return 9 + byte(i), needsFD, nil
		}
		if n < nFD {
			return 9 + byte(i), needsFD, can.ErrInvalidMsgLen
		}
	}
	return 0, needsFD, can.ErrInvalidMsgLen
}
