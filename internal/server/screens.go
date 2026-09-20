package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"thistlebbs/internal/ansi"
	"thistlebbs/internal/store"
)

// errLogout unwinds from the main menu back to the front gate.
var errLogout = errors.New("logout")

// errQuit terminates the session (hang up) when returned from the main menu.
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
			case errors.Is(err, errLogout):
				continue // back to the gate
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

func (s *Session) loginFlow() (*store.User, error) {
	for {
		s.header("Existing User Login", s.ruleTitle("login", nil))
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
		if err == nil && bcrypt.CompareHashAndPassword(
			[]byte(u.PasswordHash), []byte(password)) == nil {
			s.print("\n" + ansi.Paint(ansi.BrightGreen,
				"\u2714  Welcome back, "+u.Username+"!\n\n"))
			return u, nil
		}
		s.err("Login incorrect.  Press any key to try again, or Ctrl-D to quit.\n" +
			"(Just press Enter at the username prompt to give up.)")
	}
}

func (s *Session) registerFlow() (*store.User, error) {
	s.header("New Account Registration", s.ruleTitle("register", nil))
	s.drawRule("")
	s.print(ansi.Paint(ansi.BrightBlack,
		"Entering your name or location is optional. Passwords are stored hashed.\n\n"))

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

	s.print("\n" + ansi.Paint(ansi.BrightGreen,
		"\u2714  Account created!  Welcome, "+user.Username+"!\n\n"))
	return user, nil
}

// -- the main menu ----------------------------------------------------------

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
			case "logout":
				s.print(ansi.Paint(ansi.Yellow, "Logged out. Bye, "+s.user.Username+"!\n\n"))
				return errLogout
			case "quit":
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
		s.print(ansi.Paint(ansi.Yellow, "  [P]rofile\n"))
		s.print(ansi.Paint(ansi.Yellow, "  [L]ogout\n"))
		s.print(ansi.Paint(ansi.BrightRed, "  [Q]uit / hang up\n"))
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
		case "l":
			s.print(ansi.Paint(ansi.Yellow, "Logged out. Bye, "+s.user.Username+"!\n\n"))
			return errLogout
		case "q":
			return errQuit
		default:
			s.err(fmt.Sprintf("'%s' is not a command. Try R, P, L or Q.", choice))
		}
	}
}

// -- reading the board ------------------------------------------------------

func (s *Session) readBoard() error {
	for {
		threads, err := s.cfg.Store.Threads(s.board.ID, 1000, 0)
		if err != nil {
			return err
		}
		total := len(threads)

		pager := NewPager(s.contentHeight() - 9)
		if pager.contentHeight < 4 {
			pager.contentHeight = 4
		}

		if total == 0 {
			pager.Add(ansi.Paint(ansi.Yellow, "  No threads yet.  Press N to start the first one!"))
		} else {
			now := s.now()
			for i, t := range threads {
				line := fmt.Sprintf(" %2d. %s", i+1, clip(t.Title, s.contentWidth()-44))
				line = ansi.Paint(ansi.BrightWhite, line)
				meta := fmt.Sprintf("(%s, %d %s, %s)",
					t.Author, t.Replies, repliesWord(t.Replies), timeAgo(now, t.LastActive))
				line += ansi.Paint(ansi.BrightBlack, "  "+meta)
				pager.Add(line)
			}
		}

		s.header(fmt.Sprintf("Message Board - page %d of %d", pager.Page()+1, pager.TotalPages()), s.ruleTitle("board", nil))
		s.print(pager.Render(s.contentWidth()))

		nav := buildNav(
			paintNav(ansi.Green, "[#] open thread"),
			paintNav(ansi.Green, "[N]ew thread"),
			func() string {
				if pager.CanNext() {
					return paintNav(ansi.Yellow, "[N]ext")
				}
				return ""
			}(),
			func() string {
				if pager.CanPrev() {
					return paintNav(ansi.Yellow, "[P]rev")
				}
				return ""
			}(),
			func() string {
				if pager.CanPrev() {
					return paintNav(ansi.Yellow, "[T]op")
				}
				return ""
			}(),
			paintNav(ansi.Red, "[Q]uit"),
		)
		s.print(nav + "\n")
		s.print("\n")
		choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, "> "), true)
		if err != nil {
			return err
		}
		switch {
		case choice == "q":
			return nil
		case choice == "n":
			if err := s.newThread(); err != nil {
				return err
			}
		case choice == ">" || choice == "":
			pager.Next()
		case choice == "<" || choice == "p":
			pager.Prev()
		case choice == "t":
			pager.Top()
		default:
			if n, err := strconv.Atoi(choice); err == nil {
				if n >= 1 && n <= total {
					if err := s.viewThread(threads[n-1].ID); err != nil {
						return err
					}
				} else {
					s.err("No such thread number.")
				}
			} else {
				s.err(fmt.Sprintf("'%s' is not a command.", choice))
			}
		}
	}
}

func repliesWord(n int64) string {
	if n == 1 {
		return "reply"
	}
	return "replies"
}

func wrappedBody(p store.Post, width int) []string {
	return ansi.Wrap(p.Body, width-4)
}

// postPages returns how many in-message pages a body spans for a given budget
// of viewable lines (always >= 1).
func postPages(body []string, budget int) int {
	if len(body) == 0 || budget < 1 {
		return 1
	}
	return (len(body) + budget - 1) / budget
}

// postBlock renders one page of a single post: header, rule, wrapped body
// lines that fit this page, and footer. The surrounding blank lines are the
// caller's responsibility.
func postBlock(p store.Post, now int64, body []string, sub, budget int) []string {
	if budget < 1 {
		budget = 1
	}
	start := sub * budget
	if start > len(body) {
		start = len(body)
	}
	end := start + budget
	if end > len(body) {
		end = len(body)
	}

	lines := []string{
		ansi.Paint(ansi.BrightCyan, fmt.Sprintf("  #%d", p.ID)) + "  " +
			ansi.Paint(ansi.BrightWhite, p.Author) +
			ansi.Paint(ansi.BrightBlack, "  "+timeAgo(now, p.CreatedAt)),
		"  " + ansi.Bold + ansi.FG(ansi.Cyan) + "--" + ansi.Reset,
	}
	for _, l := range body[start:end] {
		lines = append(lines, "    "+ansi.Paint(ansi.BrightWhite, l))
	}
	lines = append(lines, ansi.Paint(ansi.BrightBlack, "    -- Posted by "+p.Author))
	return lines
}

// postMeta describes the current position and size of the thread, hinting that
// a number jumps to a specific post.
func postMeta(pos, n, pages, sub int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Post %d of %d  |  %d %s, %d %s",
		pos+1, n, n, messagesWord(n), n-1, repliesWord(int64(n-1)))
	if pages > 1 {
		fmt.Fprintf(&sb, "  |  msg page %d of %d", sub+1, pages)
	}
	sb.WriteString("  |  type a number to jump")
	return sb.String()
}

func messagesWord(n int) string {
	if n == 1 {
		return "message"
	}
	return "messages"
}

func postNavNext(sub, pages, pos, n int) string {
	if sub < pages-1 || pos < n-1 {
		return "[N]ext"
	}
	return ""
}

func postNavPrev(sub, pages, pos, n int) string {
	if sub > 0 || pos > 0 {
		return "[P]rev"
	}
	return ""
}

func postNavTop(sub, pages, pos, n int) string {
	if sub > 0 || pos > 0 {
		return "[T]op"
	}
	return ""
}

func postNavBottom(sub, pages, pos, n int) string {
	if sub < pages-1 || pos < n-1 {
		return "[B]ottom"
	}
	return ""
}

// viewThreadFrame is the fixed number of lines a view thread page renders
// around the post block: the rule title, a blank line after it, a blank line
// before it, the post meta line, and a blank line below it. The last blank
// line separates the status line from the menu that follows.
const viewThreadFrame = 5

// viewThread shows a thread one message at a time. Enter / n / '>' move to the
// next page of a long message or on to the next message; p step back;
// t and b jump to the first and last message; typing a number jumps straight
// to that post. The first post is shown on entry. Long messages are paged
// internally so a single post never overflows the screen.
func (s *Session) viewThread(threadID int64) error {
	pos, sub := 0, 0
	jumpLast := false
	for {
		posts, err := s.cfg.Store.Posts(threadID)
		if err != nil {
			return err
		}
		t, err := s.cfg.Store.ThreadByID(threadID)
		if err != nil {
			return err
		}
		if len(posts) == 0 {
			return nil
		}
		n := len(posts)

		if jumpLast {
			pos, sub = n-1, 0
			jumpLast = false
		}
		if pos < 0 {
			pos = 0
		}
		if pos >= n {
			pos = n - 1
		}

		contentHeight := s.contentHeight() - 10
		if contentHeight < 4 {
			contentHeight = 4
		}
		// The block also carries header, rule, and footer lines, so a full
		// page ends with the meta line on the last content row.
		budget := contentHeight - viewThreadFrame - 3
		if budget < 1 {
			budget = 1
		}

		body := wrappedBody(posts[pos], s.contentWidth())
		pages := postPages(body, budget)
		if sub < 0 {
			sub = 0
		}
		if sub >= pages {
			sub = pages - 1
		}
		block := postBlock(posts[pos], s.now(), body, sub, budget)
		pad := contentHeight - (len(block) + viewThreadFrame)
		if pad < 0 {
			pad = 0
		}

		meta := postMeta(pos, n, pages, sub)
		tmpl := s.cfg.menus["view_thread"]
		if tmpl != nil && tmpl.HasPostTemplate() {
			s.paint()
			vars := map[string]string{
				"title": t.Title,
				"meta":  ansi.Paint(ansi.BrightBlack, meta),
			}
			vars["rule_title"] = ansi.Paint(ansi.BrightBlue, ansi.Rule(s.contentWidth(), tmpl.RenderRule(vars)))
			s.print(strings.TrimPrefix(tmpl.Render(vars, s.contentWidth()), "\n"))
			s.print(tmpl.RenderPost(map[string]string{
				"post": strings.Join(block, "\n"),
				"pad":  strings.Repeat("\n", pad),
				"meta": ansi.Paint(ansi.BrightBlack, meta),
			}))
		} else {
			s.header(fmt.Sprintf("%s — %s", t.Title, meta))
			for _, l := range block {
				s.print(l + "\n")
			}
			s.scroll(pad)
		}

		nav := buildNav(
			paintNav(ansi.Green, "[R]eply"),
			paintNav(ansi.Yellow, postNavNext(sub, pages, pos, n)),
			paintNav(ansi.Yellow, postNavPrev(sub, pages, pos, n)),
			paintNav(ansi.Yellow, postNavTop(sub, pages, pos, n)),
			paintNav(ansi.Yellow, postNavBottom(sub, pages, pos, n)),
			paintNav(ansi.Green, "[#] jump"),
			paintNav(ansi.Red, "[Q]uit"),
		)
		s.print(nav + "\n")
		s.print("\n")
		choice, err := s.readSingleKey(ansi.Paint(ansi.BrightCyan, "> "), true)
		if err != nil {
			return err
		}
		switch {
		case choice == "q":
			return nil
		case choice == "r":
			if err := s.reply(threadID); err != nil {
				return err
			}
			jumpLast = true
		case choice == ">" || choice == "n" || choice == "":
			if sub < pages-1 {
				sub++
			} else if pos < n-1 {
				pos++
				sub = 0
			}
		case choice == "<" || choice == "p":
			if sub > 0 {
				sub--
			} else if pos > 0 {
				pos--
				sub = postPages(wrappedBody(posts[pos], s.contentWidth()), budget) - 1
			}
		case choice == "t":
			pos, sub = 0, 0
		case choice == "b":
			pos = n - 1
			sub = postPages(wrappedBody(posts[pos], s.contentWidth()), budget) - 1
		default:
			if num, err := strconv.Atoi(choice); err == nil {
				if num >= 1 && num <= n {
					pos, sub = num-1, 0
				} else {
					s.err("No such post number.")
				}
			} else {
				s.err(fmt.Sprintf("'%s' is not a command.", choice))
			}
		}
	}
}

func (s *Session) reply(threadID int64) error {
	s.header("Reply to thread", s.ruleTitle("reply", nil))
	s.print(ansi.Paint(ansi.BrightBlack,
		"Write your reply.  End the message with '.' on its own line.\n"+
			"Ctrl-C aborts.\n\n"))
	body, err := s.readText("")
	if err != nil {
		if errors.Is(err, ErrAbort) {
			s.print(ansi.Paint(ansi.Yellow, "Reply abandoned.\n"))
			return nil
		}
		return err
	}
	if body == "" {
		s.err("Reply was empty; nothing posted.")
		return nil
	}
	if _, err := s.cfg.Store.CreatePost(threadID, s.user.ID, body); err != nil {
		s.err("Could not post your reply right now.")
		return nil
	}
	s.print(ansi.Paint(ansi.BrightGreen, "\u2714  Reply posted.\n"))
	return nil
}

func (s *Session) newThread() error {
	s.header("Start a New Thread", s.ruleTitle("new_thread", nil))
	title, err := s.readLine(ansi.Paint(ansi.BrightCyan, "Title: "), false)
	if err != nil {
		return err
	}
	if title == "" {
		s.err("No title, no thread.")
		return nil
	}
	title = clip(title, 80)
	s.print("\n")
	s.print(ansi.Paint(ansi.BrightBlack,
		"Enter the first message.  End with '.' on its own line, Ctrl-C to abort.\n"))
	body, err := s.readText("")
	if err != nil {
		if errors.Is(err, ErrAbort) {
			s.print(ansi.Paint(ansi.Yellow, "Thread abandoned.\n"))
			return nil
		}
		return err
	}
	if body == "" {
		s.err("Thread had empty body; nothing posted.")
		return nil
	}
	t, _, err := s.cfg.Store.CreateThread(s.board.ID, s.user.ID, title, body)
	if err != nil {
		s.err("Could not create the thread right now.")
		return nil
	}
	s.print(ansi.Paint(ansi.BrightGreen, "\u2714  Thread created!\n"))
	return s.viewThread(t.ID)
}

// -- profile ----------------------------------------------------------------

func (s *Session) profile() error {
	for {
		threads, posts, _ := s.cfg.Store.UserStats(s.user.ID)
		s.header("Profile: "+s.user.Username, s.ruleTitle("profile", map[string]string{"user_username": s.user.Username}))
		s.drawRule("")
		u := s.user
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
	s.header("Edit Profile", s.ruleTitle("edit_profile", nil))
	s.print(ansi.Paint(ansi.BrightBlack,
		"Press Enter to keep the current value.\n\n"))

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
