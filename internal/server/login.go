package server

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"thistlebbs/internal/ansi"
	"thistlebbs/internal/store"
)

// stampLogin records a successful login (unix seconds) for the user.
func (s *Session) stampLogin(u *store.User) {
	t := s.now()
	if err := s.cfg.Store.SetLastLogin(u.ID, t); err != nil {
		s.err("Could not note your login time.")
	}
	u.LastLogin = t
}

func (s *Session) loginFlow() (*store.User, error) {
	for {
		if tmpl := s.cfg.menus["login"]; tmpl != nil {
			s.header(tmpl.RenderLocator(nil), tmpl.RenderRule(nil))
		} else {
			s.header("Existing User Login", s.ruleTitle("login", nil))
		}
		username, err := s.readLine(ansi.Paint(ansi.BrightCyan, "Username: "), false)
		if err != nil {
			return nil, err
		}
		if username == "" {
			return nil, nil
		}
		password, err := s.readLine(ansi.Paint(ansi.BrightCyan, "Password: "), true)
		if err != nil {
			return nil, err
		}

		u, err := s.cfg.Store.UserByUsername(username)
		if errors.Is(err, store.ErrNotFound) {
			if tmpl := s.cfg.menus["login"]; tmpl != nil {
				prompt := tmpl.Prompt
				if prompt == "" {
					prompt = "> "
				}
				s.print(tmpl.Render(map[string]string{"username": username}, s.contentWidth()))
				choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, prompt), false)
				if err != nil {
					return nil, err
				}
				switch tmpl.MatchInput(choice) {
				case "create":
					return s.registerFlow()
				case "quit":
					return nil, errQuit
				case "retry":
					continue
				default:
					s.err(tmpl.Errorf(choice))
					continue
				}
			}
			// Fallback: hardcoded
			s.print("\n" + ansi.Paint(ansi.Yellow, "'"+username+"' doesn't exist.\n\n"))
			s.print(ansi.Paint(ansi.BrightGreen, "  [C]reate a new account\n"))
			s.print(ansi.Paint(ansi.BrightCyan, "  [T]ry again\n"))
			s.print(ansi.Paint(ansi.BrightRed, "  [Q]uit / hang up\n"))
			choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, "\n> "), false)
			if err != nil {
				return nil, err
			}
			switch choice {
			case "c":
				return s.registerFlow()
			case "q":
				return nil, errQuit
			default:
				continue
			}
		}
		if err == nil && bcrypt.CompareHashAndPassword(
			[]byte(u.PasswordHash), []byte(password)) == nil {
			s.stampLogin(u)
			s.print("\n" + ansi.Paint(ansi.BrightGreen,
				"\u2714  Welcome back, "+u.Username+"!\n\n"))
			return u, nil
		}
		s.err("Login incorrect.  Press any key to try again, or Ctrl-D to quit.\n" +
			"(Just press Enter at the username prompt to give up.)")
	}
}

func (s *Session) registerFlow() (*store.User, error) {
	if tmpl := s.cfg.menus["register"]; tmpl != nil {
		s.header(tmpl.RenderLocator(nil), tmpl.RenderRule(nil))
		s.print(tmpl.Render(nil, s.contentWidth()))
	} else {
		s.header("New Account Registration", s.ruleTitle("register", nil))
		s.drawRule("")
		s.print(ansi.Paint(ansi.BrightBlack,
			"Entering your name or location is optional. Passwords are stored hashed.\n\n"))
	}

	username, err := s.readLine(ansi.Paint(ansi.BrightCyan, "Choose a username: "), false)
	if err != nil {
		return nil, err
	}
	if !isValidUsername(username) {
		s.err("Username must be 3-32 characters: letters, digits, _ or - only.")
		return nil, nil
	}
	if existing, err := s.cfg.Store.UserByUsername(username); err == nil && existing != nil {
		s.err("That username is already taken.")
		return nil, nil
	}

	password, err := s.readLine(ansi.Paint(ansi.BrightCyan, "Choose a password: "), true)
	if err != nil {
		return nil, err
	}
	if len(password) < 4 {
		s.err("Password must be at least 4 characters.")
		return nil, nil
	}

	location, err := s.readLine(ansi.Paint(ansi.BrightCyan, "Location (optional): "), false)
	if err != nil {
		return nil, err
	}

	bio, err := s.readLine(ansi.Paint(ansi.BrightCyan, "Bio, one line (optional): "), false)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &store.User{
		Username:     username,
		PasswordHash: string(hash),
		Location:     clip(location, 60),
		Bio:          clip(bio, 120),
		JoinedAt:     s.now(),
	}
	if err := s.cfg.Store.CreateUser(user); err != nil {
		if errors.Is(err, store.ErrUserExists) {
			s.err("That username is already taken.")
		} else {
			s.err("Account could not be created right now. Sorry.")
		}
		return nil, nil
	}
	s.stampLogin(user)

	s.print("\n" + ansi.Paint(ansi.BrightGreen,
		"\u2714  Account created!  Welcome, "+user.Username+"!\n\n"))
	return user, nil
}

// -- the main menu ----------------------------------------------------------
