package smssplitter

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectCoding(t *testing.T) {
	require.Equal(t, DataCodingGSM7, detectCoding("Hello World!"))
	require.Equal(t, DataCodingUCS2, detectCoding("سلام دنیا"))
	require.Equal(t, DataCodingUCS2, detectCoding("Hello 👋"))
}

func TestSplit_Empty(t *testing.T) {
	parts := Split("")
	require.Nil(t, parts)
}

func TestSplit_GSM7(t *testing.T) {
	text := "This is a simple test message with GSM7 chars."
	parts := Split(text)
	require.NotEmpty(t, parts)
	for _, p := range parts {
		require.True(t, len(p) <= 160) // With UDH overhead
	}
}

func TestSplit_UCS2(t *testing.T) {
	text := "این یک پیام تست است"
	parts := Split(text)
	require.NotEmpty(t, parts)
	for _, p := range parts {
		require.True(t, len(p) <= 140) // 67 UCS2 chars + UDH
	}
}

func TestChunkSeptets(t *testing.T) {
	input := []byte{0x01, 0x02, 0x03, 0x1B, 0x04, 0x05, 0x06, 0x07, 0x08}
	expected := [][]byte{{0x01, 0x02, 0x03}, {0x1B, 0x04, 0x05}, {0x06, 0x07, 0x08}}
	result := chunkSeptets(input, 3)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("chunkSeptets failed: got %v, want %v", result, expected)
	}
}

func TestPackSeptets(t *testing.T) {
	septets := []byte{0xE8, 0x32, 0x9B, 0xFD, 0x06}
	expected := []byte{0xE8, 0x32, 0x9B, 0xFD, 0x06}
	packed := packSeptets(septets)
	if len(packed) == 0 || packed[0] != expected[0] {
		t.Errorf("packSeptets failed: got %v, want prefix %v", packed, expected)
	}
}
