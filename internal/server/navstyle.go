package server

import "thistlebbs/internal/ansi"

// navColors is the single source of truth for the colour of every {nav}
// menu item, keyed by its label. Colours are non-bold to match the
// gate/main menus.
var navColors = map[string]int{
	"[#] open profile": ansi.Green,
	"[#] open thread":  ansi.Green,
	"[#] jump":         ansi.Green,
	"[N]ew thread":     ansi.Green,
	"[R]eply":          ansi.Green,
	"[N]ext":           ansi.Yellow,
	"[P]rev":           ansi.Yellow,
	"[T]op":            ansi.Yellow,
	"[B]ottom":         ansi.Yellow,
	"[Q]uit":           ansi.Red,
}

// navColor looks up a nav item's colour, falling back to plain white for
// labels not in navColors.
func navColor(label string) int {
	if col, ok := navColors[label]; ok {
		return col
	}
	return ansi.White
}
