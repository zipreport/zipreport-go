package main

import "testing"

func TestPrintable(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Plain Title", `Plain Title`},
		{"Café ünïcode", `Café ünïcode`},
		{`say "hi" \ ok`, `say "hi" \ ok`},
		{"A\x1b[2JB", `A\x1b[2JB`},
		{"D\rE\nF\tG", `D\rE\nF\tG`},
		{"x\xe2\x80\xaey", `x\u202ey`},
		{"p\a", `p\a`},
		{"bad\xffbyte", `bad\xffbyte`},
		{"\xc2\x85", `\u0085`},
		{"a\u00a0b", `a\u00a0b`},
	}
	for _, c := range cases {
		if got := printable(c.in); got != c.want {
			t.Errorf("printable(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPrintableError(t *testing.T) {
	cases := []struct{ in, want string }{
		{"unknown command \"bild\" for \"zipreport\"\n\nDid you mean this?\n\tbuild\n", "unknown command \"bild\" for \"zipreport\"\n\nDid you mean this?\n\tbuild\n"},
		{"A\x1b[2J\rB\n\tC\xff", "A\\x1b[2J\\rB\n\tC\\xff"},
		{`lit\n`, `lit\n`},
	}
	for _, c := range cases {
		if got := printableError(c.in); got != c.want {
			t.Errorf("printableError(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
