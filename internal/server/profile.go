package server

import (
	"fmt"
	"strings"
	"thistlebbs/internal/ansi"
)

func (s *Session) profile() error {
	for {
		threads, posts, _ := s.cfg.Store.UserStats(s.user.ID)
		u := s.user
		tmpl := s.cfg.menus["profile"]
		if tmpl != nil {
			vars := map[string]string{
				"user_username": u.Username,
				"user_location": orDash(u.Location),
				"user_bio":      orDash(u.Bio),
				"user_joined":   dateOf(u.JoinedAt),
				"user_threads":  fmt.Sprintf("%d", threads),
				"user_posts":    fmt.Sprintf("%d", posts),
			}
			s.header(tmpl.RenderLocator(vars), tmpl.RenderRule(vars))
			s.print(tmpl.Render(vars, s.contentWidth()))
			prompt := tmpl.Prompt
			if prompt == "" {
				prompt = "> "
			}
			choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, prompt), false)
			if err != nil {
				return err
			}
			switch tmpl.MatchInput(choice) {
			case "edit":
				if err := s.editProfile(); err != nil {
					return err
				}
			case "quit":
				return nil
			default:
				s.err(tmpl.Errorf(choice))
			}
			continue
		}
		// Fallback: hardcoded
		s.header("Profile: "+s.user.Username, s.ruleTitle("profile", map[string]string{"user_username": s.user.Username}))
		s.drawRule("")
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Username      ")+": %s\n", ansi.Paint(ansi.BrightWhite, u.Username))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Location      ")+": %s\n", ansi.Paint(ansi.BrightWhite, orDash(u.Location)))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Bio           ")+": %s\n", ansi.Paint(ansi.BrightWhite, orDash(u.Bio)))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Member since  ")+": %s\n", ansi.Paint(ansi.BrightWhite, dateOf(u.JoinedAt)))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Threads       ")+": %s\n", ansi.Paint(ansi.BrightWhite, fmt.Sprintf("%d", threads)))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Posts written ")+": %s\n", ansi.Paint(ansi.BrightWhite, fmt.Sprintf("%d", posts)))
		s.drawRule("")
		s.print(ansi.Paint(ansi.BrightBlack, "  [E]dit profile  [Q]uit\n"))
		choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, "> "), false)
		if err != nil {
			return err
		}
		switch choice {
		case "e":
			if err := s.editProfile(); err != nil {
				return err
			}
		case "q", "":
			return nil
		default:
			s.err(fmt.Sprintf("'%s' is not a command.", choice))
		}
	}
}

func (s *Session) editProfile() error {
	if tmpl := s.cfg.menus["edit_profile"]; tmpl != nil {
		s.header(tmpl.RenderLocator(nil), tmpl.RenderRule(nil))
		s.print(tmpl.Render(nil, s.contentWidth()))
	} else {
		s.header("Edit Profile", s.ruleTitle("edit_profile", nil))
		s.print(ansi.Paint(ansi.BrightBlack,
			"Press Enter to keep the current value.\n\n"))
	}

	get := func(prompt, current string) (string, error) {
		line, err := s.readLine(ansi.Paint(ansi.BrightCyan, prompt)+": ", false)
		if err != nil {
			return "", err
		}
		if line == "" {
			return current, nil
		}
		return line, nil
	}
	location, err := get("Location", s.user.Location)
	if err != nil {
		return err
	}
	bio, err := get("Bio", s.user.Bio)
	if err != nil {
		return err
	}

	if err := s.cfg.Store.UpdateProfile(s.user.ID, clip(location, 60), clip(bio, 120)); err != nil {
		s.err("Could not save the profile right now.")
		return nil
	}
	s.user.Location = location
	s.user.Bio = bio
	s.print(ansi.Paint(ansi.BrightGreen, "\u2714  Profile updated.\n"))
	return nil
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "\u2014"
	}
	return s
}
