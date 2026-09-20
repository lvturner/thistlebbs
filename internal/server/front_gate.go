package server

import (
	"fmt"
	"thistlebbs/internal/ansi"
)

func (s *Session) gateMenu() (string, error) {
	for {
		tmpl := s.cfg.menus["gate"]
		if tmpl != nil {
			s.print(tmpl.Render(nil, s.contentWidth()))
			prompt := tmpl.Prompt
			if prompt == "" {
				prompt = "> "
			}
			choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, prompt), false)
			if err != nil {
				return "", err
			}
			if action := tmpl.MatchInput(choice); action != "" {
				return action, nil
			}
			s.err(tmpl.Errorf(choice))
			continue
		}
		// Fallback: hardcoded
		s.print(ansi.Paint(ansi.BrightCyan, "  [L]ogin\n"))
		s.print(ansi.Paint(ansi.BrightGreen, "  [C]reate a new account\n"))
		s.print(ansi.Paint(ansi.BrightRed, "  [Q]uit / hang up\n"))
		choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, "\n> "), false)
		if err != nil {
			return "", err
		}
		switch choice {
		case "l":
			return actLogin, nil
		case "c":
			return actRegister, nil
		case "q":
			return actQuit, nil
		}
		s.err(fmt.Sprintf("'%s' is not a command. Try L, C or Q.", choice))
	}
}

// -- login and registration -------------------------------------------------
