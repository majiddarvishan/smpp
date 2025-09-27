package utility

import "unicode"

// IsEnglishOrEmoji returns true if s contains only ASCII English text
// (letters, digits, common punctuation/whitespace) and/or emojis.
// It rejects letters from other scripts (e.g., Arabic, Cyrillic, CJK).
func IsEnglishOrEmoji(s string) bool {
	for _, r := range s {
		if isASCIIEnglishAllowed(r) || isEmojiRune(r) {
			continue
		}
		return false
	}
	return true
}

// --- ASCII (English) ---

func isASCIIEnglishAllowed(r rune) bool {
	// Allow control whitespace: tab, newline, carriage return
	if r == '\t' || r == '\n' || r == '\r' {
		return true
	}
	// Allow ASCII printable 0x20..0x7E
	if r >= 0x20 && r <= 0x7E {
		return true
	}
	return false
}

// --- Emoji detection ---
// Note: This doesn’t rely on Go’s unicode tables being fully up-to-date;
// we include the common emoji blocks and special joiners/selectors used in emoji sequences.

func isEmojiRune(r rune) bool {
	// Zero Width Joiner for emoji sequences
	if r == 0x200D {
		return true
	}
	// Variation Selector-16 (forces emoji presentation)
	if r == 0xFE0F {
		return true
	}
	// Keycap combining mark (e.g., 1️⃣)
	if r == 0x20E3 {
		return true
	}
	// Skin tone modifiers
	if r >= 0x1F3FB && r <= 0x1F3FF {
		return true
	}
	// Regional indicator symbols (flags)
	if r >= 0x1F1E6 && r <= 0x1F1FF {
		return true
	}

	// Core emoji blocks (commonly used)
	switch {
	// Misc Symbols ☀☂★♥ etc.
	case r >= 0x2600 && r <= 0x26FF:
		return true
	// Dingbats ✂✈✔ etc.
	case r >= 0x2700 && r <= 0x27BF:
		return true
	// Emoticons 😀–🙏
	case r >= 0x1F600 && r <= 0x1F64F:
		return true
	// Misc Symbols and Pictographs 🌀–🗿
	case r >= 0x1F300 && r <= 0x1F5FF:
		return true
	// Transport and Map 🚀–🛑
	case r >= 0x1F680 && r <= 0x1F6FF:
		return true
	// Supplemental Symbols and Pictographs 🤺–🧿 etc.
	case r >= 0x1F900 && r <= 0x1F9FF:
		return true
	// Symbols and Pictographs Extended-A (Unicode 13+) 🪀–🫿
	case r >= 0x1FA70 && r <= 0x1FAFF:
		return true
	// Hearts, stars, arrows often live here too (many are covered above),
	// but we also allow some common “text heart” code points explicitly:
	case r == 0x2764: // ❤
		return true
	}

	// A few emojis live in other ranges; allow if Unicode marks categorize as Symbol (So)
	// and not a typical letter/number from another script.
	if unicode.Is(unicode.So, r) {
		return true
	}

	return false
}