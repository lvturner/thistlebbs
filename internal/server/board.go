package server

import (
	"fmt"
	"strconv"
	"strings"
	"thistlebbs/internal/ansi"
)

func (s *Session) readBoard() error {
	for {
		threads, err := s.cfg.Store.Threads(s.board.ID, 1000, 0)
		if err != nil {
			return err
		}
		total := len(threads)

		pager := NewPager(s.contentHeight() - 6)
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

		nav := buildNav(
			paintNav("[#] open thread"),
			paintNav("[N]ew thread"),
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
			func() string {
				if pager.CanPrev() {
					return paintNav("[T]op")
				}
				return ""
			}(),
			paintNav("[Q]uit"),
		)
		tmpl := s.cfg.menus["board"]
		if tmpl != nil && tmpl.HasPostTemplate() {
			s.paint()
			vars := map[string]string{
				"page":  fmt.Sprintf("%d", pager.Page()+1),
				"pages": fmt.Sprintf("%d", pager.TotalPages()),
			}
			vars["rule_title"] = ansi.Paint(ansi.BrightBlue, ansi.Rule(s.contentWidth(), tmpl.RenderRule(vars)))
			s.print(strings.TrimPrefix(tmpl.Render(vars, s.contentWidth()), "\n"))
			s.print(tmpl.RenderPost(map[string]string{
				"board": pager.Render(s.contentWidth()),
				"nav":   nav,
			}))
		} else {
			s.header(fmt.Sprintf("Message Board - page %d of %d", pager.Page()+1, pager.TotalPages()), s.ruleTitle("board", nil))
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
			case "new":
				if err := s.newThread(); err != nil {
					return err
				}
			case "next":
				pager.Next()
			case "prev":
				pager.Prev()
			case "top":
				pager.Top()
			default:
				if choice == "" {
					pager.Next()
				} else if n, err := strconv.Atoi(choice); err == nil {
					if n >= 1 && n <= total {
						if err := s.viewThread(threads[n-1].ID); err != nil {
							return err
						}
					} else {
						s.err("No such thread number.")
					}
				} else {
					s.err(tmpl.Errorf(choice))
				}
			}
		} else {
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
}

func repliesWord(n int64) string {
	if n == 1 {
		return "reply"
	}
	return "replies"
}
