package server

import (
	"fmt"
	"strconv"
	"strings"

	"thistlebbs/internal/ansi"
	"thistlebbs/internal/store"
)

// lastLoginOf renders a user's last login for the profile page, absolute.
func lastLoginOf(u *store.User) string {
	if u.LastLogin == 0 {
		return "never"
	}
	return dateOf(u.LastLogin)
}

// lastLoginAgo renders a user's last login for the user list, relative.
func lastLoginAgo(now int64, u *store.User) string {
	if u.LastLogin == 0 {
		return "never"
	}
	return timeAgo(now, u.LastLogin)
}

// users lists every registered user, most recent last login first. Typing a
// row number opens that user's read-only profile.
func (s *Session) users() error {
	for {
		users, err := s.cfg.Store.AllUsers()
		if err != nil {
			return err
		}
		total := len(users)

		pager := NewPager(s.contentHeight() - 6)
		if pager.contentHeight < 4 {
			pager.contentHeight = 4
		}

		if total == 0 {
			pager.Add(ansi.Paint(ansi.BrightBlack, "  No users yet.  The chair is empty!"))
		} else {
			now := s.now()
			for i, u := range users {
				line := fmt.Sprintf(" %2d. %s", i+1, clip(u.Username, s.contentWidth()-36))
				line = ansi.Paint(ansi.BrightWhite, line)
				line += ansi.Paint(ansi.BrightBlack, "  (last login: "+lastLoginAgo(now, &u)+")")
				pager.Add(line)
			}
		}

		nav := buildNav(
			paintNav("[#] open profile"),
			func() string {
				if pager.CanNext() {
					return paintNav("[N]ext")
				}
				return ""
			}(),
			func() string {
				if pager.CanPrev() {
					return paintNav("[P]rev")
				}
				return ""
			}(),
			paintNav("[Q]uit"),
		)
		tmpl := s.cfg.menus["users"]
		if tmpl != nil && tmpl.HasPostTemplate() {
			s.paint()
			vars := map[string]string{
				"page":  fmt.Sprintf("%d", pager.Page()+1),
				"pages": fmt.Sprintf("%d", pager.TotalPages()),
			}
			vars["rule_title"] = ansi.Paint(ansi.BrightBlue, ansi.Rule(s.contentWidth(), tmpl.RenderRule(vars)))
			s.print(strings.TrimPrefix(tmpl.Render(vars, s.contentWidth()), "\n"))
			s.print(tmpl.RenderPost(map[string]string{
				"users": pager.Render(s.contentWidth()),
				"nav":   nav,
			}))
		} else {
			s.header(fmt.Sprintf("Users - page %d of %d", pager.Page()+1, pager.TotalPages()), s.ruleTitle("users", nil))
			s.print(pager.Render(s.contentWidth()))
			s.print(nav + "\n")
		}
		s.print("\n")
		choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, "> "), true)
		if err != nil {
			return err
		}

		if tmpl != nil {
			switch tmpl.MatchInput(choice) {
			case "quit":
				return nil
			case "next":
				pager.Next()
			case "prev":
				pager.Prev()
			default:
				if n, err := strconv.Atoi(choice); err == nil {
					if n >= 1 && n <= total {
						if err := s.viewUser(&users[n-1]); err != nil {
							return err
						}
					} else {
						s.err("No such user number.")
					}
				} else {
					s.err(tmpl.Errorf(choice))
				}
			}
		} else {
			switch {
			case choice == "q":
				return nil
			case choice == ">" || choice == "":
				pager.Next()
			case choice == "<" || choice == "p":
				pager.Prev()
			default:
				if n, err := strconv.Atoi(choice); err == nil {
					if n >= 1 && n <= total {
						if err := s.viewUser(&users[n-1]); err != nil {
							return err
						}
					} else {
						s.err("No such user number.")
					}
				} else {
					s.err(fmt.Sprintf("'%s' is not a command.", choice))
				}
			}
		}
	}
}

// viewUser shows a read-only profile of the given user; q returns to the
// users list.
func (s *Session) viewUser(u *store.User) error {
	for {
		threads, posts, _ := s.cfg.Store.UserStats(u.ID)
		tmpl := s.cfg.menus["view_user"]
		if tmpl != nil {
			vars := map[string]string{
				"user_username": u.Username,
				"user_location": orDash(u.Location),
				"user_bio":      orDash(u.Bio),
				"user_joined":   dateOf(u.JoinedAt),
				"user_lastlogin": lastLoginOf(u),
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
			case "quit":
				return nil
			default:
				s.err(tmpl.Errorf(choice))
			}
			continue
		}
		// Fallback: hardcoded
		s.header("Profile: "+u.Username, s.ruleTitle("view_user", map[string]string{"user_username": u.Username}))
		s.drawRule("")
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Username      ")+": %s\n", ansi.Paint(ansi.BrightWhite, u.Username))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Location      ")+": %s\n", ansi.Paint(ansi.BrightWhite, orDash(u.Location)))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Bio           ")+": %s\n", ansi.Paint(ansi.BrightWhite, orDash(u.Bio)))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Member since  ")+": %s\n", ansi.Paint(ansi.BrightWhite, dateOf(u.JoinedAt)))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Last login    ")+": %s\n", ansi.Paint(ansi.BrightWhite, lastLoginOf(u)))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Threads       ")+": %s\n", ansi.Paint(ansi.BrightWhite, fmt.Sprintf("%d", threads)))
		s.printf("  "+ansi.Paint(ansi.BrightCyan, "Posts written ")+": %s\n", ansi.Paint(ansi.BrightWhite, fmt.Sprintf("%d", posts)))
		s.drawRule("")
		s.print(ansi.Paint(ansi.BrightBlack, "  [Q]uit\n"))
		choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, "> "), false)
		if err != nil {
			return err
		}
		switch choice {
		case "q", "":
			return nil
		default:
			s.err(fmt.Sprintf("'%s' is not a command.", choice))
		}
	}
}
