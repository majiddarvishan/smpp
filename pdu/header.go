package pdu

import (
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// Header represents PDU header.
type Header interface {
	encoding.BinaryUnmarshaler
	Length() uint32
	CommandID() CommandID
	Status() Status
	Sequence() uint32
}

type header struct {
	length    uint32
	commandID CommandID
	status    Status
	sequence  uint32
}

func (h header) Length() uint32 {
	return h.length
}

func (h header) CommandID() CommandID {
	return h.commandID
}

func (h header) Status() Status {
	return h.status
}

func (h header) Sequence() uint32 {
	return h.sequence
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface.
func (h *header) UnmarshalBinary(body []byte) error {
	h.length = binary.BigEndian.Uint32(body[:4])
	if h.length < 16 {
		return errors.New("smpp: pdu length under lower limit")
	}
	if h.length > MaxPDUSize {
		return errors.New("smpp: pdu length over upper limit")
	}
	h.commandID = CommandID(binary.BigEndian.Uint32(body[4:8]))
	h.status = Status(binary.BigEndian.Uint32(body[8:12]))
	h.sequence = binary.BigEndian.Uint32(body[12:16])
	return nil
}

func (h *header) String() string {
    var sb strings.Builder
    sb.WriteString("{\n")
    sb.WriteString(fmt.Sprintf(" length: %d\n", h.length))
    sb.WriteString(fmt.Sprintf(" commandID: %d\n", h.commandID))
    sb.WriteString(fmt.Sprintf(" status: %d\n", h.status))
    sb.WriteString(fmt.Sprintf(" sequence: %d\n", h.sequence))
    sb.WriteString("}\n")
    return sb.String()
}
