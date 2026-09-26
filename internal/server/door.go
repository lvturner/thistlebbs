package server

import (
	"net"
	"os"
	"os/user"
	"sync"
	"time"

	"thistlebbs/internal/ansi"
	"thistlebbs/internal/telnet"
)

// Door is one outbound RLOGIN door (e.g. the gOLD mINE game server). The
// door speaks BSD RLOGIN (RFC 1282): on connect the BBS sends a
// null-separated identity packet whose server-user field is
// "[TAG]<username>", Tag being this BBS's unique tag.
type Door struct {
	Name string // display name, e.g. "gOLD mINE"
	Addr string // host:port
	Tag  string // 3-char BBS tag, sent as "[TAG]<username>"
}

// doorLoginPacket builds the RFC 1282 RLOGIN identity packet this BBS sends
// straight after connecting: an empty leading string, the local OS user name,
// the door account "[TAG]<username>", and the terminal type. Door
// servers (gOLD mINE included) key RLOGIN recognition off the leading NUL
// byte, so no telnet negotiation may precede the packet.
func doorLoginPacket(tag, username, doorCode string) []byte {
	term := doorTerminalField(doorCode)
	out := make([]byte, 0, len(tag)+len(username)+len(term)+4)
	out = append(out, 0x00)
	out = append(out, rloginLocalUser()...)
	out = append(out, 0x00)
	out = append(out, "["+tag+"]"+username...)
	out = append(out, 0x00)
	out = append(out, term...)
	out = append(out, 0x00)
	return out
}

const doorTerminal = "vt100/9600"

func doorTerminalField(doorCode string) string {
	if doorCode == "" {
		return doorTerminal
	}
	return "xtrn=" + doorCode
}

// rloginLocalUser is the OS account running the BBS, used as the RLOGIN
// "local user" field (gOLD mINE displays it above its welcome banner).
func rloginLocalUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "bbs"
}

// doorEncode adapts door output for the BBS's telnet session: newlines become
// CRLF, lone carriage returns are preserved (DOS doors use them to rewind to
// column 0), and a literal 0xFF is doubled so it cannot be read as IAC.
func doorEncode(b []byte) []byte {
	out := make([]byte, 0, len(b)+len(b)/40+8)
	lastCR := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		switch c {
		case '\n':
			if !lastCR {
				out = append(out, '\r')
			}
			out = append(out, '\n')
			lastCR = false
		default:
			out = append(out, c)
			if c == telnet.IAC {
				out = append(out, telnet.IAC)
			}
			lastCR = c == '\r'
		}
	}
	return out
}

const doorDialTimeout = 10 * time.Second

// doorGames opens the games door for the current user and pumps bytes in
// both directions until the door closes. Returns nil to return to the caller.
func (s *Session) doorGames(doorCode string) error {
	d := s.cfg.Games
	if d == nil || d.Addr == "" || d.Tag == "" {
		s.err("games is not configured")
		return nil
	}

	s.header("Games")
	s.print(ansi.Paint(ansi.BrightCyan, "Connecting to "+d.Name+" ("+d.Addr+")..."))
	s.print("\n")

	conn, err := net.DialTimeout("tcp", d.Addr, doorDialTimeout)
	if err != nil {
		s.err("could not reach " + d.Name + ": " + err.Error())
		return nil
	}
	return s.doorSession(d, conn, doorCode)
}

// doorSession runs the door against an already-open connection to the door
// server. The RLOGIN identity packet is the first data sent; no telnet
// negotiation precedes it. Returns nil to return to the caller.
func (s *Session) doorSession(d *Door, conn net.Conn, doorCode string) error {
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}
	defer conn.Close()

	remotePolicy := new(telnet.Policy)
	remoteParser := new(telnet.Parser)

	// RLOGIN: send the RFC 1282 identity packet as the very first bytes on
	// the wire. gOLD mINE recognises the leading NUL and auto-logs-in (or
	// auto-registers) the "[TAG]<user>" account, skipping its in-band login.
	_, _ = conn.Write(doorLoginPacket(d.Tag, s.user.Username, doorCode))

	// kill wakes both pumps: the remote pump via the conn close, the local
	// pump via the input queue abort.
	once := sync.Once{}
	kill := func() {
		once.Do(func() {
			_ = conn.Close()
			s.in.abort()
		})
	}

	remoteDone := make(chan struct{})
	go func() {
		defer close(remoteDone)
		defer kill()
		ack := true // the server's first byte is the rlogin NUL acknowledgement
		buf := make([]byte, 16*1024)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				for _, ev := range remoteParser.Feed(buf[:n]) {
					switch ev.Kind {
					case telnet.EvData:
						data := ev.Data
						if ack {
							ack = false
							if len(data) > 0 && data[0] == 0x00 {
								data = data[1:]
							}
						}
						if len(data) > 0 {
							s.writeRaw(doorEncode(data))
						}
					default:
						if rep := remotePolicy.Handle(ev); len(rep) > 0 {
							s.writeRaw(rep)
						}
					}
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Local pump: forward keystrokes to the door. Ctrl-D (0x04) returns to
	// the BBS; Ctrl-C is passed through so in-game aborts work.
	go func() {
		defer kill()
		for {
			b, err := s.in.readByte()
			if err != nil {
				return
			}
			if b == 0x04 { // Ctrl-D: leave the door
				return
			}
			if b == telnet.IAC {
				_, _ = conn.Write([]byte{telnet.IAC, telnet.IAC})
			} else {
				_, _ = conn.Write([]byte{b})
			}
		}
	}()

	<-remoteDone
	return nil
}
