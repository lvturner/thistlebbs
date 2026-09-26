package server

import (
	"net"
	"strings"
	"testing"
	"time"

	"thistlebbs/internal/store"
	"thistlebbs/internal/telnet"
)

// newDoorSession wires a Session whose "client wire" is a loopback TCP
// connection the test can read, so the door's output to the BBS user can be
// captured.
func newDoorSession(t *testing.T) (*Session, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	other, err := ln.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	t.Cleanup(func() { _ = other.Close() })
	s := &Session{
		conn: client,
		in:   newInputQueue(),
		user: &store.User{Username: "waffle"},
	}
	return s, other
}

// listenDoor starts a loopback TCP listener standing in for the door server.
// accept blocks until a client connects.
func listenDoor(t *testing.T) (addr string, accept func() net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln.Addr().String(), func() net.Conn {
		c, err := ln.Accept()
		if err != nil {
			t.Fatalf("accept: %v", err)
		}
		return c
	}
}

// readChunk reads from c until want has appeared in the accumulated stream.
func readChunk(t *testing.T, c net.Conn, want string) string {
	t.Helper()
	var got string
	buf := make([]byte, 512)
	for {
		if len(got) > 4096 {
			t.Fatalf("read too many bytes looking for %q: %q", want, got)
		}
		n, err := c.Read(buf)
		if n > 0 {
			got += string(buf[:n])
		}
		if strings.Contains(got, want) {
			return got
		}
		if err != nil {
			t.Fatalf("read for %q: %v (have %q)", want, err, got)
		}
	}
}

func TestDoorGamesLoginAndPassthrough(t *testing.T) {
	addr, accept := listenDoor(t)
	door := &Door{Name: "gOLD mINE", Addr: addr, Tag: "FPO"}
	s, client := newDoorSession(t)
	s.cfg = &Config{Games: door}

	done := make(chan error, 1)
	go func() { done <- s.doorGames("") }()

	// The door dialed: take the connection.
	server := accept()

	// 1. On connect the door sends the RLOGIN identity packet (RFC 1282):
	// "\0<luser>\0[FPO]waffle\0vt100/9600\0". The first byte is NUL and no
	// telnet negotiation may precede it.
	pkt := readChunk(t, server, "\x00[FPO]waffle\x00")
	if pkt[0] != 0x00 {
		t.Error("expected the RLOGIN packet to start with a NUL byte")
	}
	if strings.Contains(pkt, "\xff") {
		t.Error("expected no telnet negotiation before the RLOGIN packet")
	}
	if !strings.HasSuffix(pkt, "\x00") {
		t.Error("expected the RLOGIN packet to end with a NUL byte")
	}

	// 2. Plain data from the door is re-emitted with CRLF newlines.
	_, _ = server.Write([]byte("Hello\n"))
	readChunk(t, client, "Hello\r\n")

	// 3. An escaped 0xFF (IAC IAC, a literal 0xFF byte) is re-escaped.
	_, _ = server.Write([]byte{telnet.IAC, telnet.IAC})
	readChunk(t, client, string([]byte{telnet.IAC, telnet.IAC}))

	// 4. Negotiation traffic is answered by the policy, not passed through.
	_, _ = server.Write(telnet.Cmd(telnet.WILL, telnet.OptEcho))
	readChunk(t, client, string(telnet.Cmd(telnet.DONT, telnet.OptEcho)))

	// 5. Ctrl-D leaves the door and returns to the menu.
	s.in.append([]byte{0x04})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("doorGames: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("doorGames did not return after Ctrl-D")
	}
}

func TestDoorGamesRemoteClose(t *testing.T) {
	addr, accept := listenDoor(t)
	door := &Door{Name: "gOLD mINE", Addr: addr, Tag: "FPO"}
	s, _ := newDoorSession(t)
	s.cfg = &Config{Games: door}

	done := make(chan error, 1)
	go func() { done <- s.doorGames("") }()

	// Wait for the door to connect, then drop it.
	server := accept()
	_ = server.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("doorGames: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("doorGames did not return after the door closed")
	}

	// The door's local pump must be gone: queued input is not consumed.
	s.in.append([]byte("z"))
	if b, err := s.in.readByte(); err != nil || b != 'z' {
		t.Fatalf("expected the local pump to have stopped, got %q (%v)", b, err)
	}
}

func TestDoorLoginPacket(t *testing.T) {
	pkt := doorLoginPacket("FPO", "waffle", "")
	want := []byte{0x00}
	want = append(want, rloginLocalUser()...)
	want = append(want, 0x00)
	want = append(want, '[', 'F', 'P', 'O', ']', 'w', 'a', 'f', 'f', 'l', 'e', 0x00)
	want = append(want, []byte("vt100/9600")...)
	want = append(want, 0x00)
	if string(pkt) != string(want) {
		t.Errorf("packet mismatch:\n got  %q\n want %q", pkt, want)
	}
	if pkt[0] != 0x00 {
		t.Errorf("packet must begin with NUL, got %#v", pkt[0])
	}
}

func TestDoorEncode(t *testing.T) {
	cases := []struct {
		in   []byte
		want string
	}{
		// lone CR is preserved (DOS doors rewind with CR alone)...
		{[]byte("a\rb"), "a\rb"},
		// CRLF is not doubled...
		{[]byte("a\r\nb"), "a\r\nb"},
		// bare LF becomes CRLF...
		{[]byte("a\nb"), "a\r\nb"},
		// literal 0xFF is doubled so the telnet session cannot read it as IAC...
		{[]byte{telnet.IAC, 'x'}, string([]byte{telnet.IAC, telnet.IAC, 'x'})},
	}
	for _, c := range cases {
		if got := string(doorEncode(c.in)); got != c.want {
			t.Errorf("doorEncode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDoorGamesNotConfigured(t *testing.T) {
	s, _ := newTestSession(t, 0)
	if s.cfg.Games != nil {
		t.Fatal("test session should have no games door configured")
	}
	if err := s.doorGames(""); err != nil {
		t.Fatalf("doorGames with no door: %v", err)
	}
}

func TestDoorTerminalField(t *testing.T) {
	cases := map[string]string{
		"":       "vt100/9600",
		"WORDLE": "xtrn=WORDLE",
	}
	for doorCode, want := range cases {
		if got := doorTerminalField(doorCode); got != want {
			t.Errorf("doorTerminalField(%q) = %q, want %q", doorCode, got, want)
		}
	}
}

func TestDoorLoginPacketWithDoorCode(t *testing.T) {
	pkt := doorLoginPacket("FPO", "waffle", "WORDLE")
	want := []byte{0x00}
	want = append(want, rloginLocalUser()...)
	want = append(want, 0x00)
	want = append(want, '[', 'F', 'P', 'O', ']', 'w', 'a', 'f', 'f', 'l', 'e', 0x00)
	want = append(want, []byte("xtrn=WORDLE")...)
	want = append(want, 0x00)
	if string(pkt) != string(want) {
		t.Errorf("packet mismatch:\n got  %q\n want %q", pkt, want)
	}
	if strings.Count(string(pkt), "\x00") != 4 {
		t.Errorf("packet should have four NUL-terminated fields, got %q", pkt)
	}
}

func TestDoorGamesWithDoorCode(t *testing.T) {
	addr, accept := listenDoor(t)
	door := &Door{Name: "gOLD mINE", Addr: addr, Tag: "FPO"}
	s, _ := newDoorSession(t)
	s.cfg = &Config{Games: door}

	done := make(chan error, 1)
	go func() { done <- s.doorGames(wordleDoorCode) }()

	server := accept()
	pkt := readChunk(t, server, "xtrn=WORDLE")
	if pkt[0] != 0x00 {
		t.Error("expected the RLOGIN packet to start with a NUL byte")
	}
	if strings.Contains(pkt, "\xff") {
		t.Error("expected no telnet negotiation before the RLOGIN packet")
	}
	if !strings.HasSuffix(pkt, "\x00") {
		t.Error("expected the RLOGIN packet to end with a NUL byte")
	}

	s.in.append([]byte{0x04})
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("doorGames: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("doorGames did not return after Ctrl-D")
	}
}
