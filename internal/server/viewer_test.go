package server

import (
	"testing"

	"thistlebbs/internal/ansi"
	"thistlebbs/internal/store"
)

func TestPaintNav(t *testing.T) {
	if got := paintNav(ansi.Green, ""); got != "" {
		t.Errorf("empty item: got %q, want empty", got)
	}
	got := paintNav(ansi.Yellow, "[N]ext")
	if got != ansi.Paint(ansi.Yellow, "[N]ext") {
		t.Errorf("painted item: got %q", got)
	}
}

func TestPostPages(t *testing.T) {
	cases := []struct {
		name   string
		body   []string
		budget int
		want   int
	}{
		{"empty", nil, 5, 1},
		{"one line", []string{"a"}, 5, 1},
		{"exact fit", make([]string, 5), 5, 1},
		{"one over", make([]string, 6), 5, 2},
		{"two pages exactly", make([]string, 10), 5, 2},
		{"odd remainder", make([]string, 11), 5, 3},
		{"bad budget", make([]string, 4), 0, 1},
	}
	for _, c := range cases {
		if got := postPages(c.body, c.budget); got != c.want {
			t.Errorf("%s: postPages(%d lines, budget %d) = %d; want %d",
				c.name, len(c.body), c.budget, got, c.want)
		}
	}
}

func TestPostMeta(t *testing.T) {
	if got := postMeta(0, 1, 1, 0); got != "Post 1 of 1  |  1 message, 0 replies  |  type a number to jump" {
		t.Errorf("single post: got %q", got)
	}
	want := "Post 3 of 12  |  12 messages, 11 replies  |  msg page 2 of 3  |  type a number to jump"
	if got := postMeta(2, 12, 3, 1); got != want {
		t.Errorf("multi post: got %q, want %q", got, want)
	}
}

func TestWrappedBody(t *testing.T) {
	p := store.Post{Body: "one two three four five"}
	lines := wrappedBody(p, 20)
	if len(lines) == 0 {
		t.Fatal("no wrapped lines")
	}
	for _, l := range lines {
		if len(l) > 16 {
			t.Errorf("line exceeds wrap width: %q", l)
		}
	}
}

func TestPostBlockLayout(t *testing.T) {
	body := []string{"l1", "l2", "l3", "l4", "l5", "l6", "l7", "l8"}
	block := postBlock(store.Post{ID: 7, Author: "alice"}, 0, body, 1, 3)
	if len(block) != 6 { // header, rule, 3 body lines, footer
		t.Fatalf("len(block) = %d; want 6", len(block))
	}
	if want := "    " + ansi.Paint(ansi.BrightWhite, "l4"); block[2] != want {
		t.Errorf("first body line = %q, want %q", block[2], want)
	}
	if block[5] != ansi.Paint(ansi.BrightBlack, "    -- Posted by alice") {
		t.Errorf("footer = %q", block[5])
	}
}
