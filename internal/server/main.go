package server

import (
	"fmt"
	"thistlebbs/internal/ansi"
)

func (s *Session) mainMenu() error {
	for {
		tmpl := s.cfg.menus["main"]
		if tmpl != nil {
			s.header(tmpl.RenderLocator(nil), tmpl.RenderRule(nil))
			s.print(tmpl.Render(nil, s.contentWidth()))
			prompt := tmpl.Prompt
			if prompt == "" {
				prompt = "> "
			}
			choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, prompt), false)
			if err != nil {
				return err
			}
			action := tmpl.MatchInput(choice)
			switch action {
			case "read":
				if err := s.readBoard(); err != nil {
					return err
				}
			case "profile":
				if err := s.profile(); err != nil {
					return err
				}
			case "users":
				if err := s.users(); err != nil {
					return err
				}
			case "games":
				if err := s.gamesListMenu(); err != nil {
					return err
				}
			case "logout":
				s.print(ansi.Paint(ansi.Yellow, "Logged off. Goodbye, "+s.user.Username+"!\n\n"))
				return errQuit
			default:
				s.err(tmpl.Errorf(choice))
			}
			continue
		}
		// Fallback: hardcoded
		s.header("Main Menu", "")
		s.drawRule("")
		s.print(ansi.Paint(ansi.BrightCyan, "  [R]ead board\n"))
		s.print(ansi.Paint(ansi.BrightMagenta, "  [G]ames\n"))
		s.print(ansi.Paint(ansi.BrightWhite, "  [U]sers\n"))
		s.print("\n")
		s.print(ansi.Paint(ansi.BrightYellow, "  [P]rofile\n"))
		s.print(ansi.Paint(ansi.BrightYellow, "  [L]ogoff\n"))
		choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, "\n> "), false)
		if err != nil {
			return err
		}
		switch choice {
		case "r":
			if err := s.readBoard(); err != nil {
				return err
			}
		case "p":
			if err := s.profile(); err != nil {
				return err
			}
		case "u":
			if err := s.users(); err != nil {
				return err
			}
		case "g":
			if err := s.gamesListMenu(); err != nil {
				return err
			}
		case "l":
			s.print(ansi.Paint(ansi.BrightYellow, "Logged off. Goodbye, "+s.user.Username+"!\n\n"))
			return errQuit
		default:
			s.err(fmt.Sprintf("'%s' is not a command. Try R, G, U, P or L.", choice))
		}
	}
}

// -- reading the board ------------------------------------------------------
