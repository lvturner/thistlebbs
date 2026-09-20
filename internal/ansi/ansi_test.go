package ansi

import (
	"strings"
	"testing"
)

func TestBlockLettersBasic(t *testing.T) {
	rows := BlockLetters("HI")
	if rows[0] == "" {
		t.Fatal("expected non-empty rows")
	}
	// Each row should be 15 chars (7+1+7)
	for i, r := range rows {
		if len(r) != 15 {
			t.Errorf("row %d: expected 15 chars, got %d: %q", i, len(r), r)
		}
	}
}

func TestBlockLettersEmpty(t *testing.T) {
	rows := BlockLetters("")
	for i, r := range rows {
		if r != "" {
			t.Errorf("row %d: expected empty, got %q", i, r)
		}
	}
}

func TestBlockLettersUnknownRune(t *testing.T) {
	rows := BlockLetters("T Z")
	// T is known, space is ignored, Z is unknown
	if rows[0] == "" {
		t.Fatal("expected non-empty rows for T")
	}
}

func TestBannerLines(t *testing.T) {
	lines := BannerLines("TEST")
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}
	for i, l := range lines {
		if !strings.Contains(l, "\x1b[") {
			t.Errorf("line %d: expected ANSI codes, got %q", i, l)
		}
	}
}

func TestRuleWithTitle(t *testing.T) {
	r := Rule(40, "Hello")
	if len(r) != 40 {
		t.Fatalf("expected 40 chars, got %d", len(r))
	}
	if !strings.Contains(r, "Hello") {
		t.Fatalf("expected title in rule, got %q", r)
	}
}

func TestRuleNoTitle(t *testing.T) {
	r := Rule(20, "")
	if len(r) != 20 {
		t.Fatalf("expected 20 chars, got %d", len(r))
	}
	for _, c := range r {
		if c != '-' {
			t.Fatalf("expected all dashes, got %q", r)
		}
	}
}

func TestPad(t *testing.T) {
	if got := Pad("hi", 5); got != "hi   " {
		t.Fatalf("expected 'hi   ', got %q", got)
	}
	if got := Pad("hello", 3); got != "hel" {
		t.Fatalf("expected 'hel', got %q", got)
	}
}

func TestCenter(t *testing.T) {
	got := Center("hi", 8)
	if len(got) != 8 {
		t.Fatalf("expected 8 chars, got %d", len(got))
	}
	if !strings.Contains(got, "hi") {
		t.Fatalf("expected 'hi' in centered string, got %q", got)
	}
}

func TestWrapSimple(t *testing.T) {
	lines := Wrap("hello world foo bar", 12)
	for _, l := range lines {
		if len(l) > 12 {
			t.Errorf("line too long: %q (%d)", l, len(l))
		}
	}
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
}

func TestWrapEmpty(t *testing.T) {
	lines := Wrap("", 10)
	if len(lines) != 0 {
		t.Fatalf("expected 0 lines, got %d", len(lines))
	}
}

func TestSGR(t *testing.T) {
	got := SGR(1, 31)
	if got != "\x1b[1;31m" {
		t.Fatalf("expected \\x1b[1;31m, got %q", got)
	}
}

func TestFG(t *testing.T) {
	got := FG(Red)
	if got != "\x1b[31m" {
		t.Fatalf("expected \\x1b[31m, got %q", got)
	}
}

func TestBrightFG(t *testing.T) {
	got := FG(BrightRed)
	if got != "\x1b[1;31m" {
		t.Fatalf("expected \\x1b[1;31m, got %q", got)
	}
}

func TestBG(t *testing.T) {
	got := BG(Blue)
	if got != "\x1b[44m" {
		t.Fatalf("expected \\x1b[44m, got %q", got)
	}
}

func TestPaint(t *testing.T) {
	got := Paint(Red, "hi")
	if !strings.HasPrefix(got, "\x1b[") || !strings.HasSuffix(got, "\x1b[0m") {
		t.Fatalf("expected ANSI-wrapped string, got %q", got)
	}
}

func TestExpandTagsReset(t *testing.T) {
	got := ExpandTags("{reset}")
	if got != Reset {
		t.Fatalf("expected %q, got %q", Reset, got)
	}
}

func TestExpandTagsBold(t *testing.T) {
	got := ExpandTags("{bold}hello{reset}")
	want := Bold + "hello" + Reset
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExpandTagsItalic(t *testing.T) {
	got := ExpandTags("{italic}hi{reset}")
	if got != Italic+"hi"+Reset {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsUnderscore(t *testing.T) {
	got := ExpandTags("{underline}hi{reset}")
	if got != Underline+"hi"+Reset {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsForeground(t *testing.T) {
	got := ExpandTags("{red}r{reset}")
	if got != FG(Red)+"r"+Reset {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsBrightForeground(t *testing.T) {
	got := ExpandTags("{brightCyan}hi{reset}")
	if got != FG(BrightCyan)+"hi"+Reset {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsBackground(t *testing.T) {
	got := ExpandTags("{bg:blue}hi{reset}")
	if got != BG(Blue)+"hi"+Reset {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsBrightBackground(t *testing.T) {
	got := ExpandTags("{bg:brightRed}hi{reset}")
	if got != BG(BrightRed)+"hi"+Reset {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsMixedCase(t *testing.T) {
	got := ExpandTags("{Red}r{RESET}")
	if got != FG(Red)+"r"+Reset {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsNoTag(t *testing.T) {
	got := ExpandTags("hello world")
	if got != "hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsUnknownStripped(t *testing.T) {
	got := ExpandTags("a{foobar}b")
	if got != "ab" {
		t.Fatalf("expected 'ab', got %q", got)
	}
}

func TestExpandTagsUnterminated(t *testing.T) {
	got := ExpandTags("hello {bold world")
	if got != "hello {bold world" {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsEmpty(t *testing.T) {
	got := ExpandTags("")
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestExpandTagsMultiple(t *testing.T) {
	got := ExpandTags("{bold}{cyan}hi{reset} {red}bye{reset}")
	want := Bold + FG(Cyan) + "hi" + Reset + " " + FG(Red) + "bye" + Reset
	if got != want {
		t.Fatalf("got %q", got)
	}
}
