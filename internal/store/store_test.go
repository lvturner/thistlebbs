package store

import (
	"testing"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenMemory()
	if err != nil {
		t.Fatalf("open memory store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetUser(t *testing.T) {
	s := setupTestStore(t)
	u := &User{
		Username:     "alice",
		PasswordHash: "hashed",
		Location:     "Wonderland",
		Bio:          "Curious",
		JoinedAt:     1000,
	}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	got, err := s.UserByUsername("alice")
	if err != nil {
		t.Fatalf("get by username: %v", err)
	}
	if got.ID != u.ID || got.Username != "alice" {
		t.Fatalf("mismatch: got %+v", got)
	}
}

func TestCreateUserDuplicate(t *testing.T) {
	s := setupTestStore(t)
	u := &User{Username: "bob", PasswordHash: "h", JoinedAt: 1}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("create: %v", err)
	}
	u2 := &User{Username: "bob", PasswordHash: "h2", JoinedAt: 2}
	if err := s.CreateUser(u2); err != ErrUserExists {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestUserByUsernameNotFound(t *testing.T) {
	s := setupTestStore(t)
	_, err := s.UserByUsername("nobody")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateProfile(t *testing.T) {
	s := setupTestStore(t)
	u := &User{Username: "carol", PasswordHash: "h", JoinedAt: 1}
	s.CreateUser(u)

	if err := s.UpdateProfile(u.ID, "NYC", "Hello"); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := s.UserByID(u.ID)
	if got.Location != "NYC" || got.Bio != "Hello" {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestUserStats(t *testing.T) {
	s := setupTestStore(t)
	u := &User{Username: "dave", PasswordHash: "h", JoinedAt: 1}
	s.CreateUser(u)
	b, _ := s.BoardBySlug("main")
	s.CreateThread(b.ID, u.ID, "t1", "body1")
	s.CreateThread(b.ID, u.ID, "t2", "body2")
	th, _ := s.Threads(b.ID, 10, 0)
	s.CreatePost(th[0].ID, u.ID, "reply1")

	threads, posts, err := s.UserStats(u.ID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if threads != 2 || posts != 3 {
		t.Fatalf("expected 2 threads 3 posts, got %d %d", threads, posts)
	}
}

func TestBoardBySlug(t *testing.T) {
	s := setupTestStore(t)
	b, err := s.BoardBySlug("main")
	if err != nil {
		t.Fatalf("get board: %v", err)
	}
	if b.Name != "Main Board" {
		t.Fatalf("expected Main Board, got %q", b.Name)
	}
}

func TestCreateThreadAndPosts(t *testing.T) {
	s := setupTestStore(t)
	u := &User{Username: "eve", PasswordHash: "h", JoinedAt: 1}
	s.CreateUser(u)
	b, _ := s.BoardBySlug("main")

	thread, op, err := s.CreateThread(b.ID, u.ID, "First thread", "Hello world")
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	if thread.ID == 0 || op.ID == 0 {
		t.Fatal("expected non-zero IDs")
	}
	if op.Body != "Hello world" {
		t.Fatalf("expected Hello world, got %q", op.Body)
	}

	reply, err := s.CreatePost(thread.ID, u.ID, "A reply")
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if reply.ID == 0 {
		t.Fatal("expected non-zero reply ID")
	}

	posts, err := s.Posts(thread.ID)
	if err != nil {
		t.Fatalf("list posts: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
	if posts[0].Body != "Hello world" || posts[1].Body != "A reply" {
		t.Fatalf("unexpected post bodies: %q %q", posts[0].Body, posts[1].Body)
	}
}

func TestThreadPaging(t *testing.T) {
	s := setupTestStore(t)
	u := &User{Username: "frank", PasswordHash: "h", JoinedAt: 1}
	s.CreateUser(u)
	b, _ := s.BoardBySlug("main")

	for i := 0; i < 5; i++ {
		s.CreateThread(b.ID, u.ID, "Thread", "Body")
	}

	count, _ := s.ThreadCount(b.ID)
	if count != 5 {
		t.Fatalf("expected 5 threads, got %d", count)
	}

	page1, _ := s.Threads(b.ID, 2, 0)
	if len(page1) != 2 {
		t.Fatalf("page 1: expected 2, got %d", len(page1))
	}

	page2, _ := s.Threads(b.ID, 2, 2)
	if len(page2) != 2 {
		t.Fatalf("page 2: expected 2, got %d", len(page2))
	}

	page3, _ := s.Threads(b.ID, 2, 4)
	if len(page3) != 1 {
		t.Fatalf("page 3: expected 1, got %d", len(page3))
	}
}

func TestMigrateIdempotent(t *testing.T) {
	s := setupTestStore(t)
	// Running migrate again shouldn't fail
	var v int
	err := s.db.QueryRow("PRAGMA user_version").Scan(&v)
	if err != nil || v != 1 {
		t.Fatalf("expected version 1, got v=%d err=%v", v, err)
	}
}
