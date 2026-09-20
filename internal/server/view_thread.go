package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"thistlebbs/internal/ansi"
	"thistlebbs/internal/store"
)

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
