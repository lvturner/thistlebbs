package server

import (
	"fmt"

	"thistlebbs/internal/ansi"
)

const wordleDoorCode = "WORDLE"

func (s *Session) gamesListMenu() error {
	for {
		tmpl := s.cfg.menus["games"]
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
			switch tmpl.MatchInput(choice) {
			case "wordle":
				if err := s.doorGames(wordleDoorCode); err != nil {
					return err
				}
			case "more":
				if err := s.doorGames(""); err != nil {
					return err
				}
			case "quit":
				return nil
			default:
				s.err(tmpl.Errorf(choice))
			}
			continue
		}
		s.header("Games List", "")
		s.drawRule("")
		s.print(ansi.Paint(ansi.BrightCyan, "  [W]ordle\n"))
		s.print(ansi.Paint(ansi.BrightMagenta, "  [M]ore Games\n"))
		s.print("\n")
		s.print(ansi.Paint(ansi.BrightRed, "  [Q]uit\n"))
		choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, "\n> "), false)
		if err != nil {
			return err
		}
		switch choice {
		case "w":
			if err := s.doorGames(wordleDoorCode); err != nil {
				return err
			}
		case "m":
			if err := s.doorGames(""); err != nil {
				return err
			}
		case "q":
			return nil
		default:
			s.err(fmt.Sprintf("'%s' is not a command. Try W, M or Q.", choice))
		}
	}
}
