package server

import (
	"errors"
	"thistlebbs/internal/ansi"
	"thistlebbs/internal/store"
)

// errQuit terminates the session (hang up) when returned from the main menu
// (logoff).
var errQuit = errors.New("quit")

// Menu actions returned by the front gate.
const (
	actLogin    = "login"
	actRegister = "register"
	actQuit     = "quit"
)

func (s *Session) paint() {
	s.print(ansi.ClearScreen + ansi.Move(2, 1) + ansi.Reset)
}

// header repaints the screen and draws the locator line under a rule.
// When ruleTitle is omitted the board name is used; pass an explicit title
// (including "" for a plain unadorned rule) to override.
func (s *Session) header(locator string, ruleTitle ...string) {
	s.paint()
	title := s.boardName()
	if len(ruleTitle) > 0 {
		title = ruleTitle[0] // explicit override (may be "" for plain rule)
	}
	s.print(ansi.Paint(ansi.BrightBlue, ansi.Rule(s.contentWidth(), title)) + "\n")
	s.print("\n")
	who := "not logged in"
	if s.user != nil {
		who = "Caller: " + s.user.Username
	}
	s.print(ansi.Paint(ansi.BrightCyan, locator) + "   " + ansi.Paint(ansi.BrightBlack, who) + "\n\n")
}

// ruleTitle returns the rendered rule title for the named menu template, or ""
// (plain rule) when the template is absent or declares no rule text.
func (s *Session) ruleTitle(name string, vars map[string]string) string {
	if tmpl := s.cfg.menus[name]; tmpl != nil {
		return tmpl.RenderRule(vars)
	}
	return ""
}

func (s *Session) err(msg string) {
	s.print(ansi.Paint(ansi.Red, "* "+msg+"\n"))
}

// -- public entry -----------------------------------------------------------

func (s *Session) run() error {
	s.print(ansi.AltScreenOn)
	s.drawBanner()
	for {
		action, err := s.gateMenu()
		if err != nil {
			return err
		}
		switch action {
		case actQuit:
			return nil
		}
		if action != actLogin && action != actRegister {
			return nil
		}

		var user *store.User
		if action == actLogin {
			user, err = s.loginFlow()
		} else {
			user, err = s.registerFlow()
		}
		if err != nil {
			return err
		}
		if user == nil {
			continue // back to the gate
		}

		s.user = user
		if err := s.mainMenu(); err != nil {
			s.user = nil
			switch {
			case errors.Is(err, errQuit):
				return nil // hang up
			default:
				return err
			}
		}
		s.user = nil
	}
}

// -- the front gate ---------------------------------------------------------
