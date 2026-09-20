package server

import "testing"

func TestPadLeft(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"single line", "hello", " hello"},
		{"multi line", "a\nb", " a\n b"},
		{"trailing newline", "a\n", " a\n"},
		{"leading newline", "\n> ", " \n > "},
		{"empty lines", "a\n\nb", " a\n \n b"},
	}
	for _, c := range cases {
		if got := (&Session{}).padLeft(c.in); got != c.want {
			t.Errorf("%s: padLeft(%q) = %q; want %q", c.name, c.in, got, c.want)
		}
	}
}
