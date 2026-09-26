package server

import (
	"testing"

	"thistlebbs/internal/store"
)

// newTestSession wires up a bare Session against an in-memory store. s.closed
// is set so every wire write is a no-op (no live conn), while readLine /
// readSingleKey still drain the input queue. now is stubbed to a fixed clock.
func newTestSession(t *testing.T, now int64) (*Session, *store.Store) {
	t.Helper()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("open memory store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	s := &Session{
		in:  newInputQueue(),
		cfg: &Config{Store: st},
		now: func() int64 { return now },
	}
	s.closed.Store(true)
	return s, st
}

func TestRegisterFlowStampsLogin(t *testing.T) {
	const now = int64(1_700_000_000)
	s, st := newTestSession(t, now)

	// Inputs, each line terminated by CR:
	// username, password, location (empty), bio (empty).
	s.in.append([]byte("alice\rhunter2\r\r\r"))

	u, err := s.registerFlow()
	if err != nil {
		t.Fatalf("registerFlow: %v", err)
	}
	if u == nil {
		t.Fatal("expected a user, got nil")
	}
	if u.JoinedAt != now {
		t.Fatalf("expected joined_at %d, got %d", now, u.JoinedAt)
	}
	if u.LastLogin != now {
		t.Fatalf("expected last_login %d stamped on registration, got %d", now, u.LastLogin)
	}

	got, err := st.UserByID(u.ID)
	if err != nil {
		t.Fatalf("user by id: %v", err)
	}
	if got.LastLogin == 0 {
		t.Fatal("expected last_login persisted in store, got 0 (never)")
	}
}
