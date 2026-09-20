package server

import (
	"strings"

	"thistlebbs/internal/ansi"
)

// Pager paginates a list of pre-built lines to fit a fixed content area.
// The caller renders the header and handles input; the Pager owns the
// content lines, page math, and rendering of the current page.
type Pager struct {
	lines         []string
	contentHeight int
	page          int
}

// NewPager creates a Pager that fits content into contentHeight terminal lines.
func NewPager(contentHeight int) *Pager {
	if contentHeight < 1 {
		contentHeight = 1
	}
	return &Pager{contentHeight: contentHeight}
}

// Add appends lines to the pager.
func (p *Pager) Add(lines ...string) {
	p.lines = append(p.lines, lines...)
}

// SetLines replaces all lines and resets to page 0.
func (p *Pager) SetLines(lines []string) {
	p.lines = lines
	p.page = 0
}

// TotalPages returns how many pages the content spans (always ≥ 1).
func (p *Pager) TotalPages() int {
	n := len(p.lines)
	if n == 0 {
		return 1
	}
	return (n + p.contentHeight - 1) / p.contentHeight
}

// Page returns the current page (0-indexed).
func (p *Pager) Page() int { return p.page }

// SetPage jumps to page n (clamped to valid range).
func (p *Pager) SetPage(n int) {
	p.page = clamp(n, 0, p.TotalPages()-1)
}

// Next advances one page (clamped).
func (p *Pager) Next() {
	if p.page < p.TotalPages()-1 {
		p.page++
	}
}

// Prev goes back one page (clamped).
func (p *Pager) Prev() {
	if p.page > 0 {
		p.page--
	}
}

// Top jumps to the first page.
func (p *Pager) Top() { p.page = 0 }

// Bottom jumps to the last page.
func (p *Pager) Bottom() { p.page = p.TotalPages() - 1 }

// CanNext reports whether there is a next page.
func (p *Pager) CanNext() bool { return p.page < p.TotalPages()-1 }

// CanPrev reports whether there is a previous page.
func (p *Pager) CanPrev() bool { return p.page > 0 }

// Render returns the lines for the current page, each padded to width.
// Empty lines fill the remaining space so the screen clears cleanly.
func (p *Pager) Render(width int) string {
	start := p.page * p.contentHeight
	end := start + p.contentHeight
	if end > len(p.lines) {
		end = len(p.lines)
	}

	var b strings.Builder
	for _, l := range p.lines[start:end] {
		b.WriteString(l)
		b.WriteString("\n")
	}
	// Fill remaining lines so old content is cleared
	remaining := p.contentHeight - (end - start)
	for i := 0; i < remaining; i++ {
		b.WriteString("\n")
	}
	return b.String()
}

// buildNav joins non-empty nav items with double-space separator.
func buildNav(items ...string) string {
	var parts []string
	for _, item := range items {
		if item != "" {
			parts = append(parts, item)
		}
	}
	return strings.Join(parts, "  ")
}

// paintNav colours a buildNav item, leaving empty items empty so buildNav
// still drops them.
func paintNav(col int, s string) string {
	if s == "" {
		return ""
	}
	return ansi.Paint(col, s)
}
