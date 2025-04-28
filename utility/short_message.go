package utility

import (
	"fmt"
)

func UnpackShortMessage(dataCoding int, shortMessage string) (error, *UserDataHeader, string) {
	dct := extractUnicode(dataCoding)
	if dct == dc_ascii_8_bit && len(shortMessage) > 160 {
		return fmt.Errorf("unpacking shortMessage failed, shortMessage length is larger than 160"), nil, ""
	}

	if dct != dc_ascii_8_bit && len(shortMessage) > 140 {
		return fmt.Errorf("unpacking shortMessage failed, shortMessage length is larger than 140"), nil, ""
	}

	udhl := uint8(shortMessage[0])

	if int(udhl) >= len(shortMessage) {
		return fmt.Errorf("unpacking shortMessage failed, UDH lenght is larger than shortMessage"), nil, ""
	}

	udh := NewUserDataHeader()
	udh.deserialize(shortMessage[1 : udhl+1])
	return nil, udh, shortMessage[1+udhl:]
}

func PackShortMessage(udh *UserDataHeader, body string, dataCoding int) (error, string) {
	var shortMessage string

	serialized_udh := udh.serialize()
	if len(serialized_udh) != 0 {
		shortMessage += string(byte(len(serialized_udh)))
		shortMessage += serialized_udh
	}

	shortMessage += body

	dct := extractUnicode(dataCoding)

	if dct == dc_ascii_8_bit && len(shortMessage) > 160 {
		return fmt.Errorf("packing shortMessage failed, shortMessage length is larger than 160"), ""
	}

	if dct != dc_ascii_8_bit && len(shortMessage) > 140 {
		return fmt.Errorf("packing shortMessage failed, shortMessage length is larger than 140"), ""
	}

	return nil, shortMessage
}
