package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// Thread is a message thread together with denormalised summary data used for
// listing (author name and reply count).
type Thread struct {
	ID         int64
	BoardID    int64
	AuthorID   int64
	Author     string
	Title      string
	CreatedAt  int64 // unix seconds
	LastActive int64 // unix seconds
	Replies    int64
}

// Post is a single message within a thread. The opening post of a thread has
// ID equal to the thread's first post.
type Post struct {
	ID              int64
	ThreadID        int64
	AuthorID        int64
	Author          string
	AuthorLocation  string
	AuthorJoinedAt  int64
	AuthorPostCount int64
	Body            string
	CreatedAt       int64
}

// Threads lists threads on a board, newest activity first, with paging.
func (s *Store) Threads(boardID int64, limit, offset int) ([]Thread, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.board_id, t.author_id, u.username, t.title,
		       t.created_at, t.last_activity_at,
		       (SELECT COUNT(*) - 1 FROM posts p WHERE p.thread_id = t.id)
		  FROM threads t
		  JOIN users u ON u.id = t.author_id
		 WHERE t.board_id = ?
		 ORDER BY t.last_activity_at DESC, t.id DESC
		 LIMIT ? OFFSET ?`, boardID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list threads: %w", err)
	}
	defer rows.Close()

	var out []Thread
	for rows.Next() {
		var t Thread
		if err := rows.Scan(&t.ID, &t.BoardID, &t.AuthorID, &t.Author, &t.Title,
			&t.CreatedAt, &t.LastActive, &t.Replies); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ThreadCount returns the number of threads on a board.
func (s *Store) ThreadCount(boardID int64) (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM threads WHERE board_id = ?`, boardID).Scan(&n)
	return n, err
}

// ThreadByID fetches a single thread.
func (s *Store) ThreadByID(id int64) (*Thread, error) {
	var t Thread
	err := s.db.QueryRow(`
		SELECT t.id, t.board_id, t.author_id, u.username, t.title,
		       t.created_at, t.last_activity_at,
		       (SELECT COUNT(*) - 1 FROM posts p WHERE p.thread_id = t.id)
		  FROM threads t JOIN users u ON u.id = t.author_id
		 WHERE t.id = ?`, id).
		Scan(&t.ID, &t.BoardID, &t.AuthorID, &t.Author, &t.Title,
			&t.CreatedAt, &t.LastActive, &t.Replies)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("thread by id: %w", err)
	}
	return &t, nil
}

// CreateThread starts a new thread with an opening post. Both inserts happen in
// one transaction.
func (s *Store) CreateThread(boardID, authorID int64, title, body string) (*Thread, *Post, error) {
	now := unixNow()
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("create thread tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO threads (board_id, author_id, title, created_at, last_activity_at)
		VALUES (?, ?, ?, ?, ?)`,
		boardID, authorID, title, now, now)
	if err != nil {
		return nil, nil, fmt.Errorf("insert thread: %w", err)
	}
	threadID, err := res.LastInsertId()
	if err != nil {
		return nil, nil, fmt.Errorf("thread id: %w", err)
	}

	postRes, err := tx.Exec(`
		INSERT INTO posts (thread_id, author_id, body, created_at)
		VALUES (?, ?, ?, ?)`,
		threadID, authorID, body, now)
	if err != nil {
		return nil, nil, fmt.Errorf("insert opening post: %w", err)
	}
	postID, err := postRes.LastInsertId()
	if err != nil {
		return nil, nil, fmt.Errorf("post id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit thread: %w", err)
	}

	t := &Thread{ID: threadID, BoardID: boardID, AuthorID: authorID,
		Title: title, CreatedAt: now, LastActive: now}
	p := &Post{ID: postID, ThreadID: threadID, AuthorID: authorID,
		Body: body, CreatedAt: now}
	return t, p, nil
}

// CreatePost appends a reply to a thread, bumping its last activity time.
func (s *Store) CreatePost(threadID, authorID int64, body string) (*Post, error) {
	now := unixNow()
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("create post tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO posts (thread_id, author_id, body, created_at)
		VALUES (?, ?, ?, ?)`,
		threadID, authorID, body, now)
	if err != nil {
		return nil, fmt.Errorf("insert post: %w", err)
	}
	postID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("post id: %w", err)
	}
	if _, err := tx.Exec(`
		UPDATE threads SET last_activity_at = ? WHERE id = ?`, now, threadID); err != nil {
		return nil, fmt.Errorf("bump thread: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit post: %w", err)
	}
	return &Post{ID: postID, ThreadID: threadID, AuthorID: authorID,
		Body: body, CreatedAt: now}, nil
}

// Posts returns every post in a thread in chronological order.
func (s *Store) Posts(threadID int64) ([]Post, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.thread_id, p.author_id, u.username,
		       COALESCE(u.location, ''), u.joined_at,
		       (SELECT COUNT(*) FROM posts WHERE author_id = u.id),
		       p.body, p.created_at
		  FROM posts p JOIN users u ON u.id = p.author_id
		 WHERE p.thread_id = ?
		 ORDER BY p.id`, threadID)
	if err != nil {
		return nil, fmt.Errorf("list posts: %w", err)
	}
	defer rows.Close()

	var out []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.ThreadID, &p.AuthorID, &p.Author,
			&p.AuthorLocation, &p.AuthorJoinedAt, &p.AuthorPostCount,
			&p.Body, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}