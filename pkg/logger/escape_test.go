package logger

import (
	"testing"
	"unicode/utf8"
)

func TestEscapeControl_Table(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "ascii", in: "GET /api 200", want: "GET /api 200"},
		{name: "cyrillic", in: "Привет", want: "Привет"},
		{name: "euro E2 candidate", in: "€", want: "€"},
		{name: "copyright C2 candidate", in: "©", want: "©"},
		{name: "nbsp C2 candidate", in: "\u00a0", want: "\u00a0"},
		{name: "backslash", in: `a\b`, want: `a\b`},
		{name: "backslash n literal", in: `C:\path\n`, want: `C:\path\n`},
		{name: "tab", in: "a\tb", want: "a\tb"},
		{name: "lf", in: "a\nb", want: `a\nb`},
		{name: "cr", in: "a\rb", want: `a\rb`},
		{name: "double lf", in: "x\n\ny", want: `x\n\ny`},
		{name: "nul", in: "\x00", want: `\x00`},
		{name: "ansi", in: "\x1b[31m", want: `\x1b[31m`},
		{name: "del", in: "\x7f", want: `\x7f`},
		{name: "x01", in: "\x01", want: `\x01`},
		{name: "x1f", in: "\x1f", want: `\x1f`},
		{name: "nel", in: "\u0085", want: `\u0085`},
		{name: "u0080", in: "\u0080", want: `\u0080`},
		{name: "u009f", in: "\u009f", want: `\u009f`},
		{name: "line separator", in: "\u2028", want: `\u2028`},
		{name: "paragraph separator", in: "\u2029", want: `\u2029`},
		{name: "u2027", in: "\u2027", want: "\u2027"},
		{name: "cf u202a", in: "\u202a", want: "\u202a"},
		{name: "cf u2066", in: "\u2066", want: "\u2066"},
		{name: "invalid utf8", in: "\xff\xfe", want: "\xff\xfe"},
		{name: "truncated c2", in: "\xc2", want: "\xc2"},
		{name: "truncated e2", in: "\xe2\x80", want: "\xe2\x80"},
		{name: "invalid utf8 with lf", in: "\xff\n\xc2", want: "\xff\\n\xc2"},
		{name: "mixed", in: "Привет\n€\u2028end", want: `Привет\n€\u2028end`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := escapeControl(c.in); got != c.want {
				t.Fatalf("escapeControl(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestEscapeControl_AllRunes Полнота набора: пропущенный быстрым путём кандидат — неэкранированный символ.
func TestEscapeControl_AllRunes(t *testing.T) {
	for r := rune(0); r <= utf8.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}

		in := "a" + string(r) + "b"
		got := escapeControl(in)

		if !isEscapeSetRune(r) {
			if got != in {
				t.Fatalf("rune %U: escapeControl(%q) = %q, want unchanged", r, in, got)
			}

			continue
		}

		if bad, ok := findControl(got); ok {
			t.Fatalf("rune %U: result %q contains control %U", r, got, bad)
		}
	}
}

func TestEscapeControl_NoAllocsForCleanString(t *testing.T) {
	for _, s := range []string{"GET /api/v1/items 200 12ms", "Привет, мир € ©\tтаб"} {
		var got string
		allocs := testing.AllocsPerRun(100, func() {
			got = escapeControl(s)
		})

		if allocs != 0 {
			t.Fatalf("escapeControl(%q): %v allocs, want 0", s, allocs)
		}

		if got != s {
			t.Fatalf("escapeControl(%q) = %q, want unchanged", s, got)
		}
	}
}

// isEscapeSetRune Набор из контракта, записанный независимо от реализации: Cc кроме \t, U+2028, U+2029.
func isEscapeSetRune(r rune) bool {
	switch {
	case r == '\t':
		return false
	case r <= 0x1f, r >= 0x7f && r <= 0x9f:
		return true
	default:
		return r == 0x2028 || r == 0x2029
	}
}

// findControl Ищет в результате символ набора: байты < 0x20 кроме \t, 0x7f, руны U+0080–U+009F, U+2028, U+2029.
func findControl(s string) (rune, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 && s[i] != '\t' || s[i] == 0x7f {
			return rune(s[i]), true
		}
	}

	for _, r := range s {
		if isEscapeSetRune(r) {
			return r, true
		}
	}

	return 0, false
}
