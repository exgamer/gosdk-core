package logger

import (
	"strings"
	"unicode/utf8"
)

const hexDigits = "0123456789abcdef"

// escapeControl Заменяет управляющие символы (Cc: U+0000–U+001F, U+007F–U+009F) и U+2028/U+2029
// экранированной записью, чтобы запись лога оставалась одной строкой. \t и \ не меняются.
// \n → `\n`, \r → `\r`, прочие ASCII → `\xHH`, U+0080–U+009F, U+2028, U+2029 → `\uHHHH`
// (hex в нижнем регистре, как strconv.Quote). Некорректный UTF-8 переносится байт в байт.
// Без кандидатов возвращает s без копии.
func escapeControl(s string) string {
	start := firstControl(s)
	if start < 0 {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + 16)
	b.WriteString(s[:start])

	for i := start; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if isControl(r) {
			writeEscaped(&b, r)
		} else {
			b.WriteString(s[i : i+size])
		}

		i += size
	}

	return b.String()
}

// firstControl Возвращает индекс первого экранируемого символа или -1.
// Руну декодирует только на байтах-кандидатах: остальные байты управляющий символ не начинают.
func firstControl(s string) int {
	for i := 0; i < len(s); i++ {
		if !isCandidate(s[i]) {
			continue
		}

		r, _ := utf8.DecodeRuneInString(s[i:])
		if isControl(r) {
			return i
		}
	}

	return -1
}

// isCandidate Байт, с которого может начинаться экранируемый символ:
// ASCII-управляющий кроме \t, 0xC2 (U+0080–U+00BF), 0xE2 (U+2028/U+2029).
func isCandidate(c byte) bool {
	return c < 0x20 && c != '\t' || c == 0x7f || c == 0xc2 || c == 0xe2
}

func isControl(r rune) bool {
	return r < 0x20 && r != '\t' || r >= 0x7f && r <= 0x9f || r == 0x2028 || r == 0x2029
}

func writeEscaped(b *strings.Builder, r rune) {
	switch {
	case r == '\n':
		b.WriteString(`\n`)
	case r == '\r':
		b.WriteString(`\r`)
	case r < utf8.RuneSelf:
		b.WriteString(`\x`)
		b.WriteByte(hexDigits[r>>4])
		b.WriteByte(hexDigits[r&0xf])
	default:
		b.WriteString(`\u`)
		for shift := 12; shift >= 0; shift -= 4 {
			b.WriteByte(hexDigits[r>>shift&0xf])
		}
	}
}
