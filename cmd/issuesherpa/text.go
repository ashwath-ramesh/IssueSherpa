package main

import (
	"strings"
	"unicode/utf8"
)

func sanitizeTerminalText(value string) string {
	var b strings.Builder
	b.Grow(len(value))

	for i := 0; i < len(value); {
		if skip := terminalEscapeSequenceLength(value[i:]); skip > 0 {
			i += skip
			continue
		}

		r, size := utf8.DecodeRuneInString(value[i:])
		if r == utf8.RuneError && size == 1 {
			i++
			continue
		}

		switch {
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteByte(' ')
		case r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f):
		default:
			b.WriteRune(r)
		}
		i += size
	}

	return strings.TrimSpace(b.String())
}

func terminalEscapeSequenceLength(value string) int {
	if len(value) == 0 {
		return 0
	}

	switch value[0] {
	case 0x1b:
		if len(value) == 1 {
			return 1
		}
		switch value[1] {
		case '[':
			return consumeCSI(value, 2)
		case ']':
			return consumeStringTerminated(value, 2)
		case 'P', 'X', '^', '_':
			return consumeStringTerminated(value, 2)
		default:
			if len(value) >= 2 {
				return 2
			}
			return 1
		}
	case 0x9b:
		return consumeCSI(value, 1)
	case 0x9d, 0x90, 0x98, 0x9e, 0x9f:
		return consumeStringTerminated(value, 1)
	default:
		return 0
	}
}

func consumeCSI(value string, start int) int {
	for i := start; i < len(value); i++ {
		if value[i] >= 0x40 && value[i] <= 0x7e {
			return i + 1
		}
	}
	return len(value)
}

func consumeStringTerminated(value string, start int) int {
	for i := start; i < len(value); i++ {
		if value[i] == 0x07 {
			return i + 1
		}
		if value[i] == 0x1b && i+1 < len(value) && value[i+1] == '\\' {
			return i + 2
		}
	}
	return len(value)
}
