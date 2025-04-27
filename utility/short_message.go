package smssplitter

import (
	"fmt"

	"github.com/majiddarvishan/smpp/pdu"
)

const (
	concatenated_sm_8bit_ref  uint8 = 0x00
	concatenated_sm_16bit_ref uint8 = 0x08
)

type MultiPartData struct {
	Ref   uint16 // Concatenation reference number
	Total uint8  // Total number of segments
	Seq   uint8  // Sequence number of this segment
}

type UserDataHeader struct {
	informationElements map[uint8]string
}

func NewUserDataHeader() *UserDataHeader {
	return &UserDataHeader{
		informationElements: make(map[uint8]string),
	}
}

func (udh *UserDataHeader) deserialize(buf string) error {
	var headerLen uint8 = 2

	for {
		if len(buf) < int(headerLen) {
			break
		}

		iei := uint8(buf[0])
		iedl := uint8(buf[1])

		if int(iedl) > len(buf)-int(headerLen) {
			return fmt.Errorf("user_data_header IEDL is bigger than available buf")
		}

		ied := buf[headerLen : headerLen+iedl]

		if (iei == concatenated_sm_8bit_ref && len(ied) != 3) || (iei == concatenated_sm_16bit_ref && len(ied) != 4) {
			return fmt.Errorf("MultiPartData length in UDH is invalid")
		}

		udh.informationElements[iei] = ied

		if len(buf) <= int(headerLen+iedl) {
			break
		}
		buf = string(buf[headerLen+iedl])
	}

	return nil
}

func (udh *UserDataHeader) serialize() string {
	var buf []byte

	for tag, val := range udh.informationElements {
		buf = append(buf, uint8(tag))
		buf = append(buf, uint8(len(val)))
		buf = append(buf, []byte(val)...)
	}
	return string(buf)
}

func (udh *UserDataHeader) SetMultiPartData(mpd MultiPartData) {
	if mpd.Ref > 0xFF {
		var val []byte
		val = append(val, uint8((mpd.Ref>>8)&0xFF))
		val = append(val, uint8((mpd.Ref>>0)&0xFF))
		val = append(val, mpd.Total)
		val = append(val, mpd.Seq)

		udh.informationElements[concatenated_sm_16bit_ref] = string(val)
	} else {
		var val []byte
		val = append(val, uint8(mpd.Ref))
		val = append(val, mpd.Total)
		val = append(val, mpd.Seq)

		udh.informationElements[concatenated_sm_8bit_ref] = string(val)
	}
}

func (udh *UserDataHeader) GetMultiPartData() MultiPartData {
	val, ok := udh.informationElements[concatenated_sm_8bit_ref]
	if ok {
		return MultiPartData{
			Ref:   uint16(val[0]),
			Total: val[1],
			Seq:   val[2],
		}
	}

	val, ok = udh.informationElements[concatenated_sm_16bit_ref]
	if ok {
		return MultiPartData{
			Ref:   uint16(val[0]<<8 | val[1]),
			Total: val[2],
			Seq:   val[3],
		}
	}

	return MultiPartData{
		Ref:   0,
		Total: 1,
		Seq:   1,
	}
}

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
