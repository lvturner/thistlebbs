package telnet

import (
	"testing"
)

func TestParserDataOnly(t *testing.T) {
	var p Parser
	events := p.Feed([]byte("hello"))
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EvData {
		t.Fatalf("expected EvData, got %d", events[0].Kind)
	}
	if string(events[0].Data) != "hello" {
		t.Fatalf("expected 'hello', got %q", events[0].Data)
	}
}

func TestParserDoubleIAC(t *testing.T) {
	var p Parser
	events := p.Feed([]byte{IAC, IAC, 'x'})
	if len(events) != 1 || events[0].Kind != EvData {
		t.Fatalf("expected 1 EvData, got %v", events)
	}
	if string(events[0].Data) != "\xffx" {
		t.Fatalf("expected escaped IAC + x, got %q", events[0].Data)
	}
}

func TestParserWillWontDoDont(t *testing.T) {
	var p Parser
	events := p.Feed([]byte{
		IAC, WILL, 1,
		IAC, WONT, 1,
		IAC, DO, 3,
		IAC, DONT, 3,
	})
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}
	expected := []EventKind{EvWill, EvWont, EvDo, EvDont}
	for i, ev := range events {
		if ev.Kind != expected[i] {
			t.Errorf("event %d: expected %d, got %d", i, expected[i], ev.Kind)
		}
	}
}

func TestParserSubneg(t *testing.T) {
	var p Parser
	// NAWS subneg: IAC SB NAWS 0 80 0 24 IAC SE
	events := p.Feed([]byte{
		IAC, SB, OptNAWS, 0, 80, 0, 24, IAC, SE,
	})
	if len(events) != 1 || events[0].Kind != EvSubneg {
		t.Fatalf("expected 1 EvSubneg, got %v", events)
	}
	if events[0].Opt != OptNAWS {
		t.Fatalf("expected NAWS, got %d", events[0].Opt)
	}
	if len(events[0].Data) != 4 {
		t.Fatalf("expected 4 bytes payload, got %d", len(events[0].Data))
	}
}

func TestParserSubnegEscapedIAC(t *testing.T) {
	var p Parser
	// payload contains IAC IAC (escaped 0xFF) inside SB
	events := p.Feed([]byte{
		IAC, SB, 1, IAC, IAC, IAC, SE,
	})
	if len(events) != 1 || events[0].Kind != EvSubneg {
		t.Fatalf("expected 1 EvSubneg, got %v", events)
	}
	if len(events[0].Data) != 1 || events[0].Data[0] != 0xFF {
		t.Fatalf("expected single 0xFF byte, got %v", events[0].Data)
	}
}

func TestParserDataThenCommand(t *testing.T) {
	var p Parser
	events := p.Feed(append([]byte("hi"), IAC, WILL, 1))
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Kind != EvData || string(events[0].Data) != "hi" {
		t.Errorf("event 0: expected data 'hi', got %v", events[0])
	}
	if events[1].Kind != EvWill || events[1].Opt != 1 {
		t.Errorf("event 1: expected Will opt 1, got %v", events[1])
	}
}

func TestParserSpanningChunks(t *testing.T) {
	var p Parser
	// Feed IAC in one chunk, WILL in the next
	events1 := p.Feed([]byte{IAC})
	if len(events1) != 0 {
		t.Fatalf("expected 0 events from partial IAC, got %d", len(events1))
	}
	events2 := p.Feed([]byte{WILL, 5})
	if len(events2) != 1 || events2[0].Kind != EvWill || events2[0].Opt != 5 {
		t.Fatalf("expected Will opt 5, got %v", events2)
	}
}

func TestEscapeDoublesIAC(t *testing.T) {
	out := Escape([]byte{0xFF, 'A', 0xFF})
	expected := []byte{0xFF, 0xFF, 'A', 0xFF, 0xFF}
	if len(out) != len(expected) {
		t.Fatalf("expected %d bytes, got %d: %v", len(expected), len(out), out)
	}
	for i := range expected {
		if out[i] != expected[i] {
			t.Errorf("byte %d: expected %d, got %d", i, expected[i], out[i])
		}
	}
}

func TestEncodeLineCRLF(t *testing.T) {
	// \r is stripped, \n becomes \r\n
	out := EncodeLine("a\nb\rc\n")
	expected := "a\r\nbc\r\n"
	if string(out) != expected {
		t.Fatalf("expected %q, got %q", expected, string(out))
	}
}

func TestEncodeLineEscapesIAC(t *testing.T) {
	out := EncodeLine(string([]byte{0xFF, 'x'}))
	if len(out) != 3 || out[0] != 0xFF || out[1] != 0xFF || out[2] != 'x' {
		t.Fatalf("unexpected EncodeLine output: %v", out)
	}
}

func TestSubnegBuild(t *testing.T) {
	out := Subneg(OptTTYPE, []byte{TTYPEIS, 'V', 'T'})
	// IAC SB 24 0 'V' 'T' IAC SE = 8 bytes (no IAC in payload to escape)
	expected := []byte{IAC, SB, 24, 0, 'V', 'T', IAC, SE}
	if len(out) != len(expected) {
		t.Fatalf("expected %d bytes, got %d", len(expected), len(out))
	}
	for i := range expected {
		if out[i] != expected[i] {
			t.Errorf("byte %d: expected %d, got %d", i, expected[i], out[i])
		}
	}
}

func TestConcat(t *testing.T) {
	a := []byte{1, 2}
	b := []byte{3, 4, 5}
	c := Concat(a, b)
	if len(c) != 5 || c[0] != 1 || c[4] != 5 {
		t.Fatalf("unexpected concat: %v", c)
	}
}
