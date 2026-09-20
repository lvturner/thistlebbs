// Package ansi provides helpers for composing the VT100/ANSI escape sequences
// a BBS screen uses: colours, cursor motion, clear/erase operations, text
// layout primitives and a block-letter banner.
package ansi

import (
	"fmt"
	"strings"
)

// SGR attributes.
const (
	Reset     = "\x1b[0m"
	Bold      = "\x1b[1m"
	Dim       = "\x1b[2m"
	Italic    = "\x1b[3m"
	Underline = "\x1b[4m"
	Blink     = "\x1b[5m"
	Reverse   = "\x1b[7m"
	// ShowCursor / HideCursor: DECTCEM.
	ShowCursor = "\x1b[?25h"
	HideCursor = "\x1b[?25l"
	// AltScreenOn switches to the alternate screen buffer; AltScreenOff
	// restores the caller's original screen.
	AltScreenOn  = "\x1b[?1049h"
	AltScreenOff = "\x1b[?1049l"
)

// Screen primitives.
const (
	ClearScreen = "\x1b[2J"
	ClearToEOL  = "\x1b[K"
	ClearBelow  = "\x1b[J"
)

// Home moves the cursor to row 1, column 1 (both 1-based in terminals).
func Home() string { return "\x1b[H" }

// Move positions the cursor at row/col (1-based).
func Move(row, col int) string { return fmt.Sprintf("\x1b[%d;%dH", row, col) }

// MoveCol positions the cursor at a column on the current row.
func MoveCol(col int) string { return fmt.Sprintf("\x1b[%dG", col) }

// SGR builds an SGR sequence from the given codes. Empty codes reset.
func SGR(codes ...int) string {
	if len(codes) == 0 {
		return Reset
	}
	var sb strings.Builder
	sb.WriteString("\x1b[")
	for i, c := range codes {
		if i > 0 {
			sb.WriteByte(';')
		}
		fmt.Fprintf(&sb, "%d", c)
	}
	sb.WriteByte('m')
	return sb.String()
}

// DOS 16-colour palette indices, matching ANSI SGR numbering.
const (
	Black = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
	BrightBlack
	BrightRed
	BrightGreen
	BrightYellow
	BrightBlue
	BrightMagenta
	BrightCyan
	BrightWhite
)

// FG returns the SGR to select a palette colour (0..15) as foreground.
// Bright indices (8-15) are encoded as bold + the base colour rather than the
// 90-97 SGR extension. That classic bold-on-base encoding is understood by
// legacy 16-colour clients (Windows telnet, Amiga Term, DOS ANSI.SYS) which do
// not implement the 90-97 range, while still rendering as bright on modern
// terminals.
func FG(n int) string {
	if n >= 8 {
		return SGR(1, 30+(n-8))
	}
	return SGR(30 + n)
}

// BG returns the SGR to select a palette colour (0..15) as background.
// Bright indices (8-15) are encoded as blink + the base background colour rather
// than the 100-107 SGR extension. Blink-on-base is the classic encoding by
// which legacy 16-colour clients (Windows telnet, Amiga Term, DOS ANSI.SYS)
// select a bright background, since they do not implement the 100-107 range.
func BG(n int) string {
	if n >= 8 {
		return SGR(5, 40+(n-8))
	}
	return SGR(40 + n)
}

// Paint wraps s with the bold foreground colour fg and then resets.
func Paint(fg int, s string) string { return Bold + FG(fg) + s + Reset }

// Pad right-pads or truncates s to exactly w bytes (ASCII-safe).
func Pad(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

// Center centres s within width w, truncating if it does not fit.
func Center(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	left := (w - len(s)) / 2
	right := w - left - len(s)
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// Rule renders a horizontal rule. With a non-empty title the title is centred
// between dashes.
func Rule(w int, title string) string {
	if title == "" {
		return strings.Repeat("-", w)
	}
	if len(title) >= w-2 {
		return "-" + title[:w-2] + "-"
	}
	side := (w - len(title) - 2) / 2
	rem := w - 2*side - len(title) - 2
	return strings.Repeat("-", side) + " " + title + " " + strings.Repeat("-", side+rem)
}

// Wrap splits s into lines no wider than w bytes, breaking at whitespace.
// Words longer than w are hard-split. Consecutive newlines are preserved.
func Wrap(s string, w int) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		if para == "" {
			if len(out) == 0 {
				continue
			}
			out = append(out, "")
			continue
		}
		var line strings.Builder
		for _, word := range strings.Fields(para) {
			if line.Len() == 0 {
				line.WriteString(word)
				continue
			}
			if line.Len()+1+len(word) <= w {
				line.WriteByte(' ')
				line.WriteString(word)
				continue
			}
			out = append(out, line.String())
			line.Reset()
			line.WriteString(word)
		}
		if line.Len() > 0 {
			out = append(out, line.String())
		}
	}
	return out
}

// BlockLetters renders text as six rows of seven-column block letters. The 'V'
// rune expands to full-width block, and any other glyph to a space.
func BlockLetters(s string) [6]string {
	var rows [6]string
	for _, r := range s {
		g, ok := glyphs[r]
		if !ok {
			continue
		}
		for i := 0; i < 6; i++ {
			if rows[i] != "" {
				rows[i] += " "
			}
			rows[i] += g[i]
		}
	}
	return rows
}

// BannerLines renders each word in block letters on one line, colouring the
// words in a rotating palette, suitable for the opening screen.
func BannerLines(words ...string) []string {
	var out []string
	col := Cyan
	for _, w := range words {
		g := BlockLetters(w)
		for i := 0; i < 6; i++ {
			line := Bold + FG(col) + strings.ReplaceAll(g[i], "#", "\u2588") + Reset
			out = append(out, line)
		}
		col = Green
	}
	return out
}

var glyphs = map[rune][6]string{
	'T': {"#######", "   #   ", "   #   ", "   #   ", "   #   ", "   #   "},
	'H': {"#     #", "#     #", "#######", "#     #", "#     #", "#     #"},
	'I': {"#######", "   #   ", "   #   ", "   #   ", "   #   ", "#######"},
	'S': {"#######", "#      ", "#####  ", "     # ", "#     #", "###### "},
	'L': {"#      ", "#      ", "#      ", "#      ", "#      ", "#######"},
	'E': {"#######", "#      ", "#####  ", "#      ", "#      ", "#######"},
	'B': {"#####  ", "#     #", "#     #", "#####  ", "#     #", "#####  "},
}

// fgNames maps lowercase colour names to palette indices (0-15).
var fgNames = map[string]int{
	"black": Black, "red": Red, "green": Green, "yellow": Yellow,
	"blue": Blue, "magenta": Magenta, "cyan": Cyan, "white": White,
	"brightblack": BrightBlack, "brightred": BrightRed,
	"brightgreen": BrightGreen, "brightyellow": BrightYellow,
	"brightblue": BrightBlue, "brightmagenta": BrightMagenta,
	"brightcyan": BrightCyan, "brightwhite": BrightWhite,
}

// ExpandTags replaces {tag} tokens in s with ANSI escape sequences. recognised
// tags include text styles, foreground colours, and background colours (prefixed
// with bg:). Unknown tags are silently stripped.
//
//	{reset}  {bold}  {italic}  {underscore}  {dim}
//	{red}  {brightRed}  {bg:blue}  {bg:brightMagenta}  etc.
func ExpandTags(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] != '{' {
			sb.WriteByte(s[i])
			i++
			continue
		}
		end := strings.IndexByte(s[i:], '}')
		if end < 0 {
			sb.WriteByte(s[i])
			i++
			continue
		}
		tag := strings.ToLower(s[i+1 : i+end])
		i += end + 1
		if a, ok := tagAttributes[tag]; ok {
			sb.WriteString(a)
		} else if strings.HasPrefix(tag, "bg:") {
			if c, ok := fgNames[tag[3:]]; ok {
				sb.WriteString(BG(c))
			}
		} else if c, ok := fgNames[tag]; ok {
			sb.WriteString(FG(c))
		}
		// unknown tags silently stripped
	}
	return sb.String()
}

// tagAttributes maps style tags to their ANSI constants.
var tagAttributes = map[string]string{
	"reset":     Reset,
	"bold":      Bold,
	"dim":       Dim,
	"italic":    Italic,
	"underline": Underline,
}
