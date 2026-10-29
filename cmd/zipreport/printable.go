package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// printable escapes control, format and invalid UTF-8 characters so untrusted text cannot drive the terminal
func printable(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			fmt.Fprintf(&b, `\x%02x`, s[i])
		case unicode.IsPrint(r):
			b.WriteRune(r)
		default:
			q := strconv.QuoteRune(r)
			b.WriteString(q[1 : len(q)-1])
		}
		i += size
	}
	return b.String()
}

// printableError is printable for error text, but keeps line breaks and tabs (cobra's suggestions are multi-line)
func printableError(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		cells := strings.Split(line, "\t")
		for j, c := range cells {
			cells[j] = printable(c)
		}
		lines[i] = strings.Join(cells, "\t")
	}
	return strings.Join(lines, "\n")
}
