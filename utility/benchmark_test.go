package utility

import (
	"strings"
	"testing"
)

func BenchmarkSplitGSM7(b *testing.B) {
	sample := strings.Repeat("Hello, this is a GSM7 test message. ", 10) // ~390 chars
	for i := 0; i < b.N; i++ {
		SplitGSM7(sample)
	}
}

func BenchmarkSplitUCS2(b *testing.B) {
	sample := strings.Repeat("你好，这是 UCS2 测试消息。", 10) // ~90 chars, UCS2
	for i := 0; i < b.N; i++ {
		SplitUCS2(sample)
	}
}

func BenchmarkSplitAutoDetect(b *testing.B) {
	sample := strings.Repeat("Hello你好", 50) // triggers UCS2 due to Chinese
	for i := 0; i < b.N; i++ {
		Split(sample)
	}
}
