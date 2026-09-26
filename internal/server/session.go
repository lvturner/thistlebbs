package server

import (
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"thistlebbs/internal/ansi"
	"thistlebbs/internal/store"
	"thistlebbs/internal/telnet"
)

// Sentinel errors from the input layer.
var (
	// ErrAbort is returned when the user presses Ctrl-C.
	ErrAbort = errors.New("input aborted")
	// ErrClosed is returned once the connection has gone away.
	ErrClosed = errors.New("connection closed")
	// ErrHangUp is returned when the user types Ctrl-D to hang up.
	ErrHangUp = errors.New("hang up")
	// ErrQueueAbort is returned from readByte after abort; it means "the
	// current phase is over, stop reading", not that the session is done.
	ErrQueueAbort = errors.New("input queue aborted")
)

// inputQueue is a byte FIFO fed by the read loop and drained by the line
// reader. Closing the queue wakes any blocked reader with ErrClosed; abort
// wakes any blocked reader with ErrQueueAbort without closing the queue.
type inputQueue struct {
	mu      sync.Mutex
	cond    *sync.Cond
	buf     []byte
	closed  bool
	aborted bool
}

func newInputQueue() *inputQueue {
	q := &inputQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *inputQueue) append(b []byte) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.aborted = false
	q.buf = append(q.buf, b...)
	q.cond.Broadcast()
}

func (q *inputQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()
}

// abort wakes any blocked reader with ErrQueueAbort and clears the flag, so
// queued readers can be stopped mid-phase (e.g. a door closing) without
// ending the session.
func (q *inputQueue) abort() {
	q.mu.Lock()
	q.aborted = true
	q.cond.Broadcast()
	q.mu.Unlock()
}

func (q *inputQueue) readByte() (byte, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.buf) == 0 {
		if q.closed {
			return 0, ErrClosed
		}
		if q.aborted {
			q.aborted = false
			return 0, ErrQueueAbort
		}
		q.cond.Wait()
	}
	b := q.buf[0]
	q.buf = q.buf[1:]
	return b, nil
}

const (
	// screenWidth and screenHeight are the standard BBS canvas the server
	// renders to. Client window sizes (NAWS) are honoured up to these bounds.
	screenWidth  = 80
	screenHeight = 25
)

// Session is one telnet client: the wire layer plus the menu state machine.
// The read loop and the screen code run on different goroutines and talk
// through the input queue.
type Session struct {
	conn net.Conn
	cfg  *Config
	id   int64

	parser telnet.Parser
	policy telnet.Policy

	writeMu sync.Mutex
	closed  atomic.Bool

	in  *inputQueue
	now func() int64 // clock, stubbed in tests

	user    *store.User
	board   *store.Board
	width   int
	height  int
	bannerD bool
}

func newSession(conn net.Conn, cfg *Config, id int64) *Session {
	board, _ := cfg.Store.BoardBySlug("main") // migration guarantees it exists
	return &Session{
		conn:   conn,
		cfg:    cfg,
		id:     id,
		in:     newInputQueue(),
		now:    func() int64 { return unixNow() },
		board:  board,
		width:  screenWidth,
		height: screenHeight,
	}
}

// -- wire output ------------------------------------------------------------

func (s *Session) writeRaw(b []byte) {
	if s.closed.Load() || len(b) == 0 {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, _ = s.conn.Write(b)
}

// print writes text with CRLF newlines and 0xFF escaping applied.
func (s *Session) print(text string) {
	s.writeRaw(telnet.EncodeLine(s.padLeft(text)))
}

// padLeft indents every line by one space so content never touches the left
// edge of the client. The space is inserted at the start of the string and
// after each newline (except a trailing one, to avoid a stray trailing pad).
func (s *Session) padLeft(text string) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text) + strings.Count(text, "\n") + 1)
	b.WriteByte(' ')
	for i := 0; i < len(text); i++ {
		b.WriteByte(text[i])
		if text[i] == '\n' && i+1 < len(text) {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func (s *Session) printf(format string, a ...any) {
	s.print(sprintf(format, a...))
}

func (s *Session) newline() { s.print("\n") }

// scroll writes n blank lines to push content up.
func (s *Session) scroll(n int) {
	for i := 0; i < n; i++ {
		s.print("\n")
	}
}

// boardName returns the configured board name, falling back to the seed row.
func (s *Session) boardName() string {
	if s.board != nil && s.board.Name != "" {
		return s.board.Name
	}
	if s.cfg.BoardName != "" {
		return s.cfg.BoardName
	}
	return "Main Board"
}

func (s *Session) widthOr(def int) int {
	if s.width > 0 {
		return s.width
	}
	return def
}

func (s *Session) heightOr(def int) int {
	if s.height > 0 {
		return s.height
	}
	return def
}

// contentWidth returns the usable character columns after reserving a 1-char
// pad on the left and right of the screen.
func (s *Session) contentWidth() int { return s.widthOr(screenWidth) - 2 }

// contentHeight returns the usable rows after reserving a 1-char pad at the
// top and bottom of the screen.
func (s *Session) contentHeight() int { return s.heightOr(screenHeight) - 2 }

// -- read loop --------------------------------------------------------------

func (s *Session) readLoop() {
	buf := make([]byte, 1024)
	for {
		n, err := s.conn.Read(buf)
		if n > 0 {
			events := s.parser.Feed(buf[:n])
			for _, ev := range events {
				if reply := s.policy.Handle(ev); len(reply) > 0 {
					s.writeRaw(reply)
				}
				if ev.Kind == telnet.EvData {
					s.in.append(ev.Data)
				}
			}
			s.applyDims()
		}
		if err != nil {
			if !s.closed.Load() {
				s.in.close()
			}
			return
		}
	}
}

func (s *Session) applyDims() {
	if w := s.policy.Width; w > 0 && w != s.width {
		s.width = min(w, screenWidth)
	}
	if h := s.policy.Height; h > 0 && h != s.height {
		s.height = min(h, screenHeight)
	}
}

// close shuts down the session: it wakes any blocked reader and closes the
// wire.
func (s *Session) close() {
	if s.closed.Swap(true) {
		return
	}
	s.in.close()
	_ = s.conn.Close()
}

// -- line input -------------------------------------------------------------

// readLine prompts for a single line of input. The server echoes visible
// characters and handles editing (backspace, Ctrl-U) itself. When hidden is
// true, nothing is echoed (password prompts). Returns ErrAbort on Ctrl-C and
// ErrHangUp on Ctrl-D when the line is empty.
func (s *Session) readLine(prompt string, hidden bool) (string, error) {
	s.print(prompt)

	var buf []byte
	for {
		b, err := s.in.readByte()
		if err != nil {
			if errors.Is(err, ErrClosed) {
				return "", ErrClosed
			}
			return "", err
		}
		switch {
		case b == '\r':
			s.newline()
			return strings.Trim(string(buf), " \t\r\n"), nil
		case b == 0x03: // Ctrl-C
			s.newline()
			return "", ErrAbort
		case b == 0x04: // Ctrl-D
			if len(buf) == 0 {
				return "", ErrHangUp
			}
		case b == 0x15: // Ctrl-U: kill line
			if len(buf) > 0 {
				buf = buf[:0]
				s.writeRaw([]byte("\r" + ansi.ClearToEOL + string(telnet.EncodeLine(s.padLeft(prompt)))))
			}
		case b == 0x7f || b == 0x08: // DEL / BS
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				if !hidden {
					s.writeRaw([]byte("\b \b"))
				}
			}
		case b == '\t':
			continue
		case b < 0x20:
			continue
		default:
			buf = append(buf, b)
			if !hidden {
				s.writeRaw([]byte{b})
			}
		}
	}
}

// readSingleKey reads one menu keypress without waiting for Enter. Enter
// returns "" (the "next page" default in list views). Ctrl-C aborts and
// Ctrl-D hangs up. When numeric is true, a leading digit collects a
// multi-digit number terminated by Enter (used for thread selection); stray
// letters inside a number are ignored, while lone letters dispatch
// immediately like any other key.
func (s *Session) readSingleKey(prompt string, numeric bool) (string, error) {
	s.print(prompt)
	var num []byte
	for {
		b, err := s.in.readByte()
		if err != nil {
			if errors.Is(err, ErrClosed) {
				return "", ErrClosed
			}
			return "", err
		}
		switch {
		case b == '\r':
			s.newline()
			if len(num) > 0 {
				return string(num), nil
			}
			return "", nil
		case b == 0x03: // Ctrl-C
			s.newline()
			return "", ErrAbort
		case b == 0x04: // Ctrl-D
			return "", ErrHangUp
		case b == 0x7f || b == 0x08: // DEL / BS (only meaningful inside a number)
			if len(num) > 0 {
				num = num[:len(num)-1]
				s.writeRaw([]byte("\b \b"))
			}
			continue
		case b < 0x20:
			continue
		case numeric && b >= '0' && b <= '9':
			num = append(num, b)
			s.writeRaw([]byte{b})
			continue
		default:
			if len(num) > 0 {
				continue // mid-number: ignore stray keys
			}
			s.writeRaw([]byte{b})
			return lower(string(b)), nil
		}
	}
}

// readText reads a multiline message. The user ends with "." on its own line,
// or Ctrl-D. Returns ErrAbort on Ctrl-C.
func (s *Session) readText(prompt string) (string, error) {
	s.print(prompt)
	var lines []string
	for {
		line, err := s.readLine(ansi.Paint(ansi.BrightCyan, "> "), false)
		if err != nil {
			return "", err
		}
		if line == "." {
			break
		}
		lines = append(lines, line)
		if totalLen(lines) > maxMessageChars {
			s.print(ansi.Paint(ansi.Red, "* Message too long; truncating.\n"))
			lines = truncateLines(lines, maxMessageChars)
			break
		}
	}
	return strings.Join(lines, "\n"), nil
}

const maxMessageChars = 4000

func totalLen(lines []string) int {
	n := 0
	for _, l := range lines {
		n += len(l) + 1
	}
	return n
}

func truncateLines(lines []string, max int) []string {
	n := 0
	out := lines[:0]
	for _, l := range lines {
		if n+len(l)+1 > max {
			break
		}
		out = append(out, l)
		n += len(l) + 1
	}
	return out
}

// -- helper text ------------------------------------------------------------

func (s *Session) drawRule(title string) {
	s.print(ansi.Paint(ansi.BrightBlack, ansi.Rule(s.contentWidth(), title)) + "\n")
	s.print("\n")
}

func (s *Session) drawHeader() {
	if s.bannerD {
		return
	}
	s.bannerD = true
	if s.cfg.banner != "" {
		s.print(s.cfg.banner)
	} else {
		for _, l := range ansi.BannerLines("THISTLE", "BBS") {
			s.print(l + "\n")
		}
		s.print("\n")
		s.print(ansi.Paint(ansi.Yellow, "\u2591 Welcome to "+s.cfg.BoardName+" \u2591\n\n"))
	}
}

func (s *Session) drawBanner() {
	s.print(ansi.ClearScreen + ansi.Move(2, 1))
	s.drawHeader()
	s.drawRule("")
}
