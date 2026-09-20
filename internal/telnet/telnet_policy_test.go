package telnet

import (
	"testing"
)

func TestPolicyPreamble(t *testing.T) {
	p := Policy{}
	pre := p.Preamble()
	if len(pre) == 0 {
		t.Fatal("preamble should not be empty")
	}
	// Should contain: WILL SGA, DO NAWS, DO BINARY, DO TTYPE, SB TTYPE SEND
	if !contains(pre, Cmd(WILL, OptSGA)) {
		t.Error("missing WILL SGA")
	}
	if !contains(pre, Cmd(DO, OptNAWS)) {
		t.Error("missing DO NAWS")
	}
	if !contains(pre, Cmd(DO, OptBinary)) {
		t.Error("missing DO BINARY")
	}
	if !contains(pre, Cmd(DO, OptTTYPE)) {
		t.Error("missing DO TTYPE")
	}
}

func TestPolicyHandleWillSGA(t *testing.T) {
	p := Policy{}
	reply := p.Handle(Event{Kind: EvWill, Opt: OptSGA})
	if len(reply) != 3 || reply[0] != IAC || reply[1] != DO || reply[2] != OptSGA {
		t.Fatalf("expected IAC DO SGA, got %v", reply)
	}
}

func TestPolicyHandleWillEchoDuringPassword(t *testing.T) {
	p := Policy{passwordPrompt: true}
	reply := p.Handle(Event{Kind: EvWill, Opt: OptEcho})
	if len(reply) != 3 || reply[0] != IAC || reply[1] != DO || reply[2] != OptEcho {
		t.Fatalf("expected IAC DO Echo, got %v", reply)
	}
	if !p.echoAccepted {
		t.Fatal("echoAccepted should be true")
	}
}

func TestPolicyHandleWillEchoNotPassword(t *testing.T) {
	p := Policy{passwordPrompt: false}
	reply := p.Handle(Event{Kind: EvWill, Opt: OptEcho})
	if len(reply) != 3 || reply[0] != IAC || reply[1] != DONT || reply[2] != OptEcho {
		t.Fatalf("expected IAC DONT Echo, got %v", reply)
	}
}

func TestPolicyHandleDoTtype(t *testing.T) {
	p := Policy{}
	reply := p.Handle(Event{Kind: EvDo, Opt: OptTTYPE})
	if len(reply) != 3 || reply[0] != IAC || reply[1] != WILL || reply[2] != OptTTYPE {
		t.Fatalf("expected IAC WILL TTYPE, got %v", reply)
	}
}

func TestPolicyHandleDoNaws(t *testing.T) {
	p := Policy{}
	reply := p.Handle(Event{Kind: EvDo, Opt: OptNAWS})
	if len(reply) != 3 || reply[0] != IAC || reply[1] != WILL || reply[2] != OptNAWS {
		t.Fatalf("expected IAC WILL NAWS, got %v", reply)
	}
}

func TestPolicyHandleDoUnknown(t *testing.T) {
	p := Policy{}
	reply := p.Handle(Event{Kind: EvDo, Opt: 99})
	if len(reply) != 3 || reply[0] != IAC || reply[1] != WONT || reply[2] != 99 {
		t.Fatalf("expected IAC WONT 99, got %v", reply)
	}
}

func TestPolicySubnegNaws(t *testing.T) {
	p := Policy{}
	p.Handle(Event{Kind: EvSubneg, Opt: OptNAWS, Data: []byte{0, 80, 0, 24}})
	if p.Width != 80 || p.Height != 24 {
		t.Fatalf("expected 80x24, got %dx%d", p.Width, p.Height)
	}
}

func TestPolicySubnegTtypeIs(t *testing.T) {
	p := Policy{}
	p.Handle(Event{Kind: EvSubneg, Opt: OptTTYPE, Data: append([]byte{TTYPEIS}, "xterm-256color"...)})
	if p.TermType != "xterm-256color" {
		t.Fatalf("expected xterm-256color, got %q", p.TermType)
	}
}

func TestPolicySubnegTtypeSend(t *testing.T) {
	p := Policy{}
	reply := p.Handle(Event{Kind: EvSubneg, Opt: OptTTYPE, Data: []byte{TTYPESEND}})
	// Should reply SB TTYPE IS "ANSI"
	if len(reply) < 6 {
		t.Fatalf("expected subneg reply, got %v", reply)
	}
	if reply[0] != IAC || reply[1] != SB || reply[2] != OptTTYPE || reply[3] != TTYPEIS {
		t.Fatalf("unexpected reply header: %v", reply[:4])
	}
	if string(reply[4:len(reply)-2]) != "ANSI" {
		t.Fatalf("expected ANSI in reply, got %v", reply[4:len(reply)-2])
	}
}

func TestPolicyPasswordEchoRestore(t *testing.T) {
	p := Policy{}
	pe := p.PasswordEcho()
	if len(pe) != 3 || pe[0] != IAC || pe[1] != WILL || pe[2] != OptEcho {
		t.Fatalf("expected IAC WILL Echo, got %v", pe)
	}
	if !p.passwordPrompt {
		t.Fatal("passwordPrompt should be true")
	}
	re := p.RestoreEcho()
	if len(re) != 3 || re[0] != IAC || re[1] != WONT || re[2] != OptEcho {
		t.Fatalf("expected IAC WONT Echo, got %v", re)
	}
	if p.passwordPrompt {
		t.Fatal("passwordPrompt should be false")
	}
}

func TestPolicyWontDontReturnNil(t *testing.T) {
	p := Policy{}
	if r := p.Handle(Event{Kind: EvWont, Opt: 1}); r != nil {
		t.Errorf("WONT should return nil, got %v", r)
	}
	if r := p.Handle(Event{Kind: EvDont, Opt: 1}); r != nil {
		t.Errorf("DONT should return nil, got %v", r)
	}
}

func contains(haystack []byte, needle []byte) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
