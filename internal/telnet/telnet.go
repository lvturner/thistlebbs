// Package telnet implements the wire-level pieces of the telnet protocol
// (RFC 854/855) needed by a BBS server: an IAC state machine that splits the
// byte stream into plain data and negotiation traffic, plus the byte encoders
// for writing output.
package telnet

// Command and option constants (RFC 854, 855, 884, 1073).
const (
	IAC  = byte(255) // interpret as command
	DONT = byte(254)
	DO   = byte(253)
	WONT = byte(252)
	WILL = byte(251)
	SB   = byte(250) // subnegotiation begin
	SE   = byte(240) // subnegotiation end
)

const (
	OptBinary   = byte(0) // transmit binary
	OptEcho     = byte(1) // echo
	OptSGA      = byte(3) // suppress go ahead
	OptTTYPE    = byte(24)
	OptNAWS     = byte(31)
	OptTermType = OptTTYPE
)

// TTYPE subnegotiation qualifiers.
const (
	TTYPEIS   = byte(0)
	TTYPESEND = byte(1)
)

// EventKind classifies a parsed telnet event.
type EventKind int

const (
	EvData EventKind = iota // plain payload bytes
	EvWill
	EvWont
	EvDo
	EvDont
	EvSubneg // an SB ... SE exchange, payload in Event.Data
)

// Event is a single parsed telnet event.
type Event struct {
	Kind EventKind
	Opt  byte   // option byte for negotiation/subnegotiation events
	Data []byte // payload for EvData and EvSubneg
}

// Parser is a streaming IAC state machine. Feed copies keep their own storage,
// so the returned slice may be held across subsequent Feed calls.
type Parser struct {
	st    parserState
	events []Event
	cmd   byte
	sbOpt byte
	sb    []byte
}

type parserState int

const (
	stData parserState = iota
	stIAC
	stOpt
	stSB
	stSBIAC
)

// Feed digests a chunk of raw bytes and returns the events produced by it.
func (p *Parser) Feed(b []byte) []Event {
	start := len(p.events)
	var cur []byte
	flush := func() {
		if len(cur) > 0 {
			p.events = append(p.events, Event{Kind: EvData, Data: append([]byte(nil), cur...)})
			cur = cur[:0]
		}
	}
	for _, c := range b {
		switch p.st {
		case stData:
			if c == IAC {
				flush()
				p.st = stIAC
			} else {
				cur = append(cur, c)
			}
		case stIAC:
			switch c {
			case IAC: // doubled IAC is a literal 0xFF byte
				cur = append(cur, IAC)
				p.st = stData
			case DO, DONT, WILL, WONT:
				p.cmd = c
				p.st = stOpt
			case SB:
				p.sbOpt = 0
				p.sb = p.sb[:0]
				p.st = stSB
			default: // NOP, GA, EOR, ... are dropped
				p.st = stData
			}
		case stOpt:
			p.events = append(p.events, Event{Kind: kindForCmd(p.cmd), Opt: c})
			p.st = stData
		case stSB:
			if p.sbOpt == 0 {
				p.sbOpt = c
			} else if c == IAC {
				p.st = stSBIAC
			} else {
				p.sb = append(p.sb, c)
			}
		case stSBIAC:
			if c == SE {
				p.events = append(p.events, Event{Kind: EvSubneg, Opt: p.sbOpt,
					Data: append([]byte(nil), p.sb...)})
				p.st = stData
			} else if c == IAC {
				p.sb = append(p.sb, IAC)
				p.st = stSB
			} else {
				p.st = stData
			}
		}
	}
	flush()
	out := make([]Event, len(p.events)-start)
	copy(out, p.events[start:])
	return out
}

func kindForCmd(c byte) EventKind {
	switch c {
	case WILL:
		return EvWill
	case WONT:
		return EvWont
	case DO:
		return EvDo
	case DONT:
		return EvDont
	}
	return EvData
}

// Cmd builds a three-byte negotiation command: IAC cmd opt.
func Cmd(cmd, opt byte) []byte {
	return []byte{IAC, cmd, opt}
}

// Subneg builds an IAC SB opt payload IAC SE sequence, escaping 0xFF bytes.
func Subneg(opt byte, payload []byte) []byte {
	out := make([]byte, 0, len(payload)+4)
	out = append(out, IAC, SB, opt)
	out = appendEscaped(out, payload)
	return append(out, IAC, SE)
}

func appendEscaped(out []byte, b []byte) []byte {
	for _, c := range b {
		out = append(out, c)
		if c == IAC {
			out = append(out, IAC)
		}
	}
	return out
}

// Escape doubles any 0xFF byte so payload may be sent verbatim.
func Escape(b []byte) []byte {
	return appendEscaped(nil, b)
}

// EncodeLine converts a string for transmission: it normalizes newlines to
// CRLF (dropping any bare CR), escapes 0xFF, and returns the raw bytes.
func EncodeLine(s string) []byte {
	out := make([]byte, 0, len(s)+len(s)/40+4)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\n':
			out = append(out, '\r', '\n')
		case '\r':
			// drop bare CR; a CR directly before \n is covered by the \n case
		default:
			out = append(out, c)
			if c == IAC {
				out = append(out, IAC)
			}
		}
	}
	return out
}