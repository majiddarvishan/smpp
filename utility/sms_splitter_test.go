package smssplitter

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectCoding(t *testing.T) {
	require.Equal(t, DataCodingGSM7, detectCoding("Hello World!"))
	require.Equal(t, DataCodingUCS2, detectCoding("سلام دنیا"))
	require.Equal(t, DataCodingUCS2, detectCoding("Hello 👋"))
}

func TestSplit_Empty(t *testing.T) {
	parts, _, _ := Split("")
	require.Nil(t, parts)
}

func TestSplit_GSM7(t *testing.T) {
	text := "This is a simple test message with GSM7 chars."
	parts, dcs, err := Split(text)
	require.NoError(t, err)
	require.Equal(t, DataCodingGSM7, dcs)
	require.NotEmpty(t, parts)
	for _, p := range parts {
		require.True(t, len(p) <= 160) // With UDH overhead
	}
}

func TestSplit_UCS2(t *testing.T) {
	text := "این یک پیام تست است"
	parts, dcs, err := Split(text)
	require.NoError(t, err)
	require.Equal(t, DataCodingUCS2, dcs)
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

func TestSplitWithUDH_OnePart_GSM7(t *testing.T) {
	text := "Hello from GSM7!"
	res, err := SplitWithUDH(text)
	require.NoError(t, err)
	require.Equal(t, DataCodingGSM7, res.Coding)
	require.Len(t, res.Bodies, 1)
    require.NotEmpty(t, res.Bodies[0], "Body payload should not be empty")
}

func TestSplitWithUDH_GSM7(t *testing.T) {
    text := strings.Repeat("Hello, this is a GSM7 test message. ", 10) // ~390 chars
	res, err := SplitWithUDH(text)
	require.NoError(t, err)
	require.Equal(t, DataCodingGSM7, res.Coding)
	require.Len(t, res.UDHs, len(res.Bodies))
	for i, udh := range res.UDHs {
		require.Len(t, udh.Pack(), 6, "UDH length")
		require.NotEmpty(t, res.Bodies[i], "Body payload should not be empty")
	}
}

func TestSplitWithUDH_OnePart_UCS2(t *testing.T) {
	text := "سلام دنیا"
	res, err := SplitWithUDH(text)
	require.NoError(t, err)
	require.Equal(t, DataCodingUCS2, res.Coding)
	require.Len(t, res.Bodies, 1)
    require.NotEmpty(t, res.Bodies[0], "Body payload should not be empty")
}

func TestSplitWithUDH_UCS2(t *testing.T) {
    text := strings.Repeat("سلام دنیا. این یک تست پیام طولانی فارسی است", 10)
	res, err := SplitWithUDH(text)
	require.NoError(t, err)
	require.Equal(t, DataCodingUCS2, res.Coding)
	require.Len(t, res.UDHs, len(res.Bodies))
	for i, udh := range res.UDHs {
		require.Len(t, udh.Pack(), 6, "UDH length")
		require.NotEmpty(t, res.Bodies[i], "Body payload should not be empty")
	}
}
