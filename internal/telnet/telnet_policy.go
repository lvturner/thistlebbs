package telnet

// Policy implements the BBS server's side of telnet option negotiation. It is
// purely reactive: it answers the client's WILL/WONT/DO/DONT and subnegotiation
// traffic, and records useful facts it learns (terminal type and window size).
// Server-initiated requests are made through dedicated helpers, so Policy stays
// a self-contained, testable unit.
type Policy struct {
	// TermType is the client's terminal type, if it announced one (TTYPE IS).
	TermType string
	// Width and Height are the client's window size from NAWS (0 = unknown).
	Width  int
	Height int

	passwordPrompt bool
	echoAccepted   bool
}

// Preamble returns the negotiation bytes to send immediately after connect:
// enable SGA, request window size and terminal type, and ask the client for its
// terminal type straight away.
func (p Policy) Preamble() []byte {
	return Concat(
		Cmd(WILL, OptSGA),
		Cmd(DO, OptNAWS),
		Cmd(DO, OptBinary),
		Cmd(DO, OptTTYPE),
		Subneg(OptTTYPE, []byte{TTYPESEND}),
	)
}

// PasswordEcho asks the client to suppress local echo for a password prompt
// (server takes over echoing, and simply chooses to echo nothing). It returns
// the negotiation bytes to send. The request is forgotten on the next prompt.
func (p *Policy) PasswordEcho() []byte {
	p.passwordPrompt = true
	p.echoAccepted = false
	return Cmd(WILL, OptEcho)
}

// RestoreEcho requests the client to resume local echo.
func (p *Policy) RestoreEcho() []byte {
	p.passwordPrompt = false
	return Cmd(WONT, OptEcho)
}

// EchoAccepted reports whether the client agreed to the most recent
// PasswordEcho request (i.e. whether it is hiding its own echo).
func (p *Policy) EchoAccepted() bool { return p.echoAccepted }

// Handle processes one incoming event and returns any reply bytes to send.
func (p *Policy) Handle(ev Event) []byte {
	switch ev.Kind {
	case EvWill:
		switch ev.Opt {
		case OptSGA:
			return Cmd(DO, OptSGA)
		case OptEcho:
			if p.passwordPrompt {
				p.echoAccepted = true
				return Cmd(DO, OptEcho)
			}
			return Cmd(DONT, OptEcho)
		default:
			return Cmd(DONT, ev.Opt)
		}
	case EvWont, EvDont:
		return nil
	case EvDo:
		switch ev.Opt {
		case OptSGA:
			return Cmd(DO, OptSGA)
		case OptBinary, OptNAWS, OptTTYPE:
			return Cmd(WILL, ev.Opt)
		case OptEcho:
			// The client wants server echo; only honour it while a password
			// prompt is active, otherwise refuse to avoid double echo.
			if p.passwordPrompt {
				return nil
			}
			return Cmd(WONT, ev.Opt)
		default:
			return Cmd(WONT, ev.Opt)
		}
	case EvSubneg:
		switch ev.Opt {
		case OptNAWS:
			if len(ev.Data) >= 4 {
				p.Width = int(ev.Data[0])<<8 | int(ev.Data[1])
				p.Height = int(ev.Data[2])<<8 | int(ev.Data[3])
			}
		case OptTTYPE:
			if len(ev.Data) == 0 {
				break
			}
			switch ev.Data[0] {
			case TTYPEIS:
				if len(ev.Data) > 1 {
					p.TermType = string(ev.Data[1:])
				}
			case TTYPESEND:
				return Subneg(OptTTYPE, append([]byte{TTYPEIS}, "ANSI"...))
			}
		}
	}
	return nil
}

// Concat joins byte slices, for composing negotiation streams.
func Concat(parts ...[]byte) []byte {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	out := make([]byte, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}