package server

import (
	"path/filepath"
	"testing"

	"thistlebbs/internal/store"
)

func TestLastLoginOf(t *testing.T) {
	u := &store.User{LastLogin: 0}
	if got := lastLoginOf(u); got != "never" {
		t.Errorf("never: got %q", got)
	}
	u = &store.User{LastLogin: 1000}
	got := lastLoginOf(u)
	if got == "never" || got == "" {
		t.Errorf("expected a date, got %q", got)
	}
}

func TestLastLoginAgo(t *testing.T) {
	u := &store.User{LastLogin: 0}
	if got := lastLoginAgo(2000, u); got != "never" {
		t.Errorf("never: got %q", got)
	}
	u = &store.User{LastLogin: 1999}
	if got := lastLoginAgo(2000, u); got != "just now" {
		t.Errorf("expected 'just now', got %q", got)
	}
}

func TestLoadUsersTemplates(t *testing.T) {
	dir := filepath.Join("..", "..", "data", "menus")

	users, err := LoadMenuTemplate(filepath.Join(dir, "users.txt"))
	if err != nil {
		t.Fatalf("load users template: %v", err)
	}
	if !users.HasPostTemplate() {
		t.Error("users template should have a per-user post section")
	}
	if users.MatchInput("q") != "quit" {
		t.Error("users template: q should quit")
	}
	if users.MatchInput(">") != "next" || users.MatchInput("<") != "prev" {
		t.Error("users template: > and < should page")
	}

	view, err := LoadMenuTemplate(filepath.Join(dir, "view_user.txt"))
	if err != nil {
		t.Fatalf("load view_user template: %v", err)
	}
	if view.MatchInput("q") != "quit" {
		t.Error("view_user template: q should quit")
	}
	got := view.RenderLocator(map[string]string{"user_username": "carol"})
	if got != "Profile: carol" {
		t.Errorf("locator: got %q", got)
	}

	main, err := LoadMenuTemplate(filepath.Join(dir, "main.txt"))
	if err != nil {
		t.Fatalf("load main template: %v", err)
	}
	if main.MatchInput("u") != "users" || main.MatchInput("users") != "users" {
		t.Error("main template: u should open the users view")
	}
}
