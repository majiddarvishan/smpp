package utility

import (
	"fmt"
)

func UnpackShortMessage(dataCoding int, shortMessage string) (*UserDataHeader, string, error) {
	dct := ExtractUnicode(dataCoding)
	if dct == DCT_Ascii_8_bit && len(shortMessage) > 160 {
		return nil, "", fmt.Errorf("unpacking shortMessage failed, shortMessage length is larger than 160")
	}

	if dct != DCT_Ascii_8_bit && len(shortMessage) > 140 {
		return nil, "", fmt.Errorf("unpacking shortMessage failed, shortMessage length is larger than 140")
	}

	udhl := uint8(shortMessage[0])

	if int(udhl) >= len(shortMessage) {
		return nil, "", fmt.Errorf("unpacking shortMessage failed, UDH lenght is larger than shortMessage")
	}

	udh := NewUserDataHeader()
	udh.deserialize(shortMessage[1 : udhl+1])
	return udh, shortMessage[1+udhl:], nil
}

func PackShortMessage(udh *UserDataHeader, body string, dataCoding int) (string, error) {
	var shortMessage string

	serialized_udh := udh.serialize()
	if len(serialized_udh) != 0 {
		shortMessage += string(byte(len(serialized_udh)))
		shortMessage += serialized_udh
	}

	shortMessage += body

	dct := ExtractUnicode(dataCoding)

	if dct == DCT_Ascii_8_bit && len(shortMessage) > 160 {
		return "", fmt.Errorf("packing shortMessage failed, shortMessage length is larger than 160")
	}

	if dct != DCT_Ascii_8_bit && len(shortMessage) > 140 {
		return "", fmt.Errorf("packing shortMessage failed, shortMessage length is larger than 140")
	}

	return shortMessage, nil
}
