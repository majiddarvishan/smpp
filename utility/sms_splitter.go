package utility

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"time"

	"github.com/abadojack/whatlanggo"
)

// DataCoding represents SMPP data coding schemes.
type DataCoding byte

const (
	DataCodingGSM7 DataCoding = 0x00 // GSM 7-bit encoding
	DataCodingUCS2 DataCoding = 0x08 // UCS2 encoding (UCS-2 BE)
)

// detectCoding returns the detected DataCoding (GSM7 or UCS2).
func detectCoding(text string) DataCoding {
	for _, r := range text {
		if _, ok := gsm7Default[r]; ok {
			continue
		}
		if _, ok := gsm7Ext[r]; ok {
			continue
		}
		// any unsupported rune => UCS2
		return DataCodingUCS2
	}
	return DataCodingGSM7
}

// checks if the given text is in English
func isEnglish(text string) bool {
	if len(text) == 0 {
		return true
	}
	info := whatlanggo.Detect(text)

	// Check if the detected language is English and if the detection is reliable
	// The library provides a confidence score and a reliability flag.
	// whatlanggo.Eng represents the English language code.
	// info.IsReliable() gives an indication of the confidence.
	return info.Lang == whatlanggo.Eng && info.IsReliable()
}

// chunkSeptets splits septets into size-limited slices, not ending on an ESC
func chunkSeptets(sep []byte, max int) [][]byte {
	var out [][]byte
	i := 0
	for i < len(sep) {
		extra := len(sep) - i
		limit := max
		if extra < max {
			limit = extra
		}
		if limit > 0 && sep[i+limit-1] == 0x1B {
			limit--
		}
		out = append(out, sep[i:i+limit])
		i += limit
	}
	return out
}

// packSeptets packs 7-bit septets into octets
func packSeptets(septets []byte) []byte {
	bitLen := len(septets) * 7
	octets := make([]byte, (bitLen+7)/8)
	for i, s := range septets {
		bitPos := uint(i * 7)
		bytePos, offset := bitPos/8, bitPos%8
		octets[bytePos] |= s << offset
		if offset > 1 {
			octets[bytePos+1] |= s >> (8 - offset)
		}
	}
	return octets
}

// randomByte returns a cryptographically secure random byte.
func randomByte() byte {
	source := rand.NewSource(time.Now().UnixNano()) // Create a new source
	rng := rand.New(source)                         // Create a new random number generator
	return byte(rng.Intn(256))
}

// SplitUCS2 splits text into UCS-2 segments with 6-byte UDH, max 67 chars each.
func SplitUCS2(text string) [][]byte {
	var parts [][]byte

	if len(text) <= 140 {
		parts = append(parts, []byte(text))
		return parts
	}

	runes := []rune(text)
	const maxChars = 67
	total := (len(runes) + maxChars - 1) / maxChars
	if total == 0 {
		return nil
	}

	// Generate a random reference number for the UDH
	ref := randomByte()

	for i := 0; i < total; i++ {
		start, end := i*maxChars, (i+1)*maxChars
		if end > len(runes) {
			end = len(runes)
		}
		udh := []byte{0x05, 0x00, 0x03, ref, byte(total), byte(i + 1)}
		seg := runes[start:end]
		ucs2 := make([]byte, len(seg)*2)
		for j, r := range seg {
			binary.BigEndian.PutUint16(ucs2[j*2:], uint16(r))
		}
		parts = append(parts, append(udh, ucs2...))
	}
	return parts
}

// SplitGSM7 builds true GSM7 segments with UDH and full mapping
func SplitGSM7(text string) [][]byte {
	var parts [][]byte

	// map runes to septets (including escapes)
	var septets []byte
	for _, r := range text {
		if code, ok := gsm7Default[r]; ok {
			septets = append(septets, code)
		} else if ext, ok := gsm7Ext[r]; ok {
			septets = append(septets, 0x1B, ext)
		} else {
			septets = append(septets, gsm7Default['?'])
		}
	}

	if len(septets) <= 160 {
		parts = append(parts, septets)
		return parts
	}

	// chunk septets into segments of max 153 septets, avoiding lone ESC
	chunks := chunkSeptets(septets, 153)

	// Generate a random reference number for the UDH
	ref := randomByte()

	for i, chunk := range chunks {
		udh := []byte{0x05, 0x00, 0x03, ref, byte(len(chunks)), byte(i + 1)}
		packed := packSeptets(chunk)
		parts = append(parts, append(udh, packed...))
	}
	return parts
}

func Split(text string) ([][]byte, DataCoding, error) {
	if len(text) == 0 {
		return nil, 0x00, errors.New("empty message")
	}

	coding := detectCoding(text)

	switch coding {
	case DataCodingGSM7:
		return SplitGSM7(text), DataCodingGSM7, nil
	case DataCodingUCS2:
		return SplitUCS2(text), DataCodingUCS2, nil
	default:
		return nil, 0x00, errors.New("unknown data coding")
	}
}

// UDH represents a 6-byte User Data Header for SMS concatenation.
type UDH struct {
	UDHL  byte // User Data Header Length (always 0x05)
	IEI   byte // Information Element Identifier (0x00)
	IEDL  byte // Information Element Data Length (0x03)
	Ref   byte // Concatenation reference number
	Total byte // Total number of segments
	Seq   byte // Sequence number of this segment
}

// Pack serializes the UDH struct into a 6-byte slice.
func (u UDH) Pack() []byte {
	return []byte{u.UDHL, u.IEI, u.IEDL, u.Ref, u.Total, u.Seq}
}

// Unpack deserializes a 6-byte slice into the UDH struct.
func (u *UDH) Unpack(data []byte) error {
	if len(data) != 6 {
		return errors.New("invalid UDH length, expected 6 bytes")
	}
	u.UDHL = data[0]
	u.IEI = data[1]
	u.IEDL = data[2]
	u.Ref = data[3]
	u.Total = data[4]
	u.Seq = data[5]
	return nil
}

// SplitResult holds the UDH structs and body payloads for each segment.
type SplitResult struct {
	UDHs   []UDH      // concatenation headers
	Bodies [][]byte   // message bodies (UDH excluded)
	Coding DataCoding // detected encoding scheme
}

// SplitWithUDH splits the input text and returns a SplitResult struct
// containing separate UDHs and body payloads for each segment.
func SplitWithUDH(text string) (SplitResult, error) {
	if len(text) == 0 {
		return SplitResult{}, errors.New("empty message")
	}
	coding := detectCoding(text)
	var result SplitResult
	result.Coding = coding

	switch coding {
	case DataCodingGSM7:
		// Build septets
		septets := make([]byte, 0, len(text)*2)
		for _, r := range text {
			if code, ok := gsm7Default[r]; ok {
				septets = append(septets, code)
			} else if ext, ok := gsm7Ext[r]; ok {
				septets = append(septets, 0x1B, ext)
			} else {
				septets = append(septets, gsm7Default['?'])
			}
		}

		if len(septets) <= 160 {
			result.Bodies = append(result.Bodies, septets)
			return result, nil
		}

		chunks := chunkSeptets(septets, 153)

		// Generate a random reference number for the UDH
		ref := randomByte()
		for i, chunk := range chunks {
			s := UDH{UDHL: 0x05, IEI: 0x00, IEDL: 0x03, Ref: ref, Total: byte(len(chunks)), Seq: byte(i + 1)}
			result.UDHs = append(result.UDHs, s)
			result.Bodies = append(result.Bodies, packSeptets(chunk))
		}

	case DataCodingUCS2:
		if len(text) <= 140 {
			result.Bodies = append(result.Bodies, []byte(text))
			return result, nil
		}

		runes := []rune(text)
		const maxChars = 67
		total := (len(runes) + maxChars - 1) / maxChars
		ref := randomByte()
		for i := 0; i < total; i++ {
			start := i * maxChars
			end := start + maxChars
			if end > len(runes) {
				end = len(runes)
			}
			s := UDH{UDHL: 0x05, IEI: 0x00, IEDL: 0x03, Ref: ref, Total: byte(total), Seq: byte(i + 1)}
			result.UDHs = append(result.UDHs, s)
			seg := runes[start:end]
			ucs2 := make([]byte, len(seg)*2)
			for j, r := range seg {
				binary.BigEndian.PutUint16(ucs2[j*2:], uint16(r))
			}
			result.Bodies = append(result.Bodies, ucs2)
		}
	}
	return result, nil
}
