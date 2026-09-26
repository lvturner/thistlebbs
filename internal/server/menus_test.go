package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadMenu(t *testing.T, name string) *MenuTemplate {
	t.Helper()
	dir := filepath.Join("..", "..", "data", "menus")
	tmpl, err := LoadMenuTemplate(filepath.Join(dir, name+".txt"))
	if err != nil {
		t.Fatalf("load %s template: %v", name, err)
	}
	return tmpl
}

// Every menu template in the directory must parse and carry the shared shape
// (a locator at minimum). The names match the fixed list the server loads.
func TestLoadAllMenuTemplates(t *testing.T) {
	names := []string{"gate", "main", "games", "profile", "edit_profile", "reply",
		"new_thread", "view_thread", "login", "register", "board", "users", "view_user"}
	for _, name := range names {
		tmpl := loadMenu(t, name)
		if tmpl.Locator == "" {
			t.Errorf("%s: missing locator", name)
		}
	}
}

func TestLoadMainTemplate(t *testing.T) {
	m := loadMenu(t, "main")
	want := map[string]string{
		"r": "read",
		"p": "profile",
		"u": "users",
		"g": "games",
		"l": "logout",
	}
	for k, action := range want {
		if got := m.MatchInput(k); got != action {
			t.Errorf("main: %q should be %q, got %q", k, action, got)
		}
	}
	// The Q/quit/hang-up binding was removed from the main menu; logoff hangs up.
	for _, k := range []string{"q", "quit", "hang"} {
		if got := m.MatchInput(k); got != "" {
			t.Errorf("main: %q should be unbound, got %q", k, got)
		}
	}
}

func TestLoadGamesTemplate(t *testing.T) {
	g := loadMenu(t, "games")
	if g.Locator != "Games List" {
		t.Errorf("games: locator = %q, want %q", g.Locator, "Games List")
	}
	want := map[string]string{
		"w":         "wordle",
		"wordle":    "wordle",
		"m":         "more",
		"more":      "more",
		"moregames": "more",
		"q":         "quit",
	}
	for k, action := range want {
		if got := g.MatchInput(k); got != action {
			t.Errorf("games: %q should be %q, got %q", k, action, got)
		}
	}
}

func TestLoadBoardTemplate(t *testing.T) {
	b := loadMenu(t, "board")
	if !b.HasPostTemplate() {
		t.Error("board template should have a per-page post section")
	}
	want := map[string]string{
		"q": "quit",
		"n": "new",
		">": "next",
		"<": "prev",
		"p": "prev",
		"t": "top",
	}
	for k, action := range want {
		if got := b.MatchInput(k); got != action {
			t.Errorf("board: %q should be %q, got %q", k, action, got)
		}
	}
	if !strings.Contains(b.post, "{nav}") {
		t.Error("board template: post section should place a {nav} token")
	}
}

func TestLoadViewThreadTemplate(t *testing.T) {
	vt := loadMenu(t, "view_thread")
	if !vt.HasPostTemplate() {
		t.Error("view_thread template should have a per-post post section")
	}
	want := map[string]string{
		"q": "quit",
		"r": "reply",
		"n": "next",
		"p": "prev",
		"t": "top",
		"b": "bottom",
	}
	for k, action := range want {
		if got := vt.MatchInput(k); got != action {
			t.Errorf("view_thread: %q should be %q, got %q", k, action, got)
		}
	}
	if !strings.Contains(vt.post, "{nav}") {
		t.Error("view_thread template: post section should place a {nav} token")
	}
}

func TestLoadLoginTemplate(t *testing.T) {
	l := loadMenu(t, "login")
	want := map[string]string{
		"c": "create",
		"t": "retry",
		"q": "quit",
	}
	for k, action := range want {
		if got := l.MatchInput(k); got != action {
			t.Errorf("login: %q should be %q, got %q", k, action, got)
		}
	}
}

func TestLoadInputScreenTemplates(t *testing.T) {
	cases := map[string][2]string{
		"register":     {"New Account Registration", "Join Us!"},
		"reply":        {"Reply to thread", ""},
		"new_thread":   {"Start a New Thread", "New Thread"},
		"edit_profile": {"Edit Profile", ""},
	}
	for name, want := range cases {
		tmpl := loadMenu(t, name)
		if tmpl.Locator != want[0] {
			t.Errorf("%s: locator = %q, want %q", name, tmpl.Locator, want[0])
		}
		if tmpl.Rule != want[1] {
			t.Errorf("%s: rule = %q, want %q", name, tmpl.Rule, want[1])
		}
	}

	// new_thread: uses the standard prompt field and has no stale custom fields.
	nt := loadMenu(t, "new_thread")
	if nt.Prompt == "" {
		t.Error("new_thread: should declare a prompt for the title input")
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "data", "menus", "new_thread.txt"))
	if err != nil {
		t.Fatalf("read new_thread.txt: %v", err)
	}
	for _, dead := range []string{"error_title", "error_empty", "error_store"} {
		if strings.Contains(string(raw), dead) {
			t.Errorf("new_thread: should not declare dead field %q", dead)
		}
	}
}
