package store

import (
	"database/sql"
	"fmt"
)

// Schema version 1: users, boards, threads and posts for the single-board era.
const schemaV1 = `
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL COLLATE NOCASE UNIQUE,
    password_hash TEXT    NOT NULL,
    location      TEXT    NOT NULL DEFAULT '',
    bio           TEXT    NOT NULL DEFAULT '',
    joined_at     INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS boards (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS threads (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    board_id         INTEGER NOT NULL REFERENCES boards(id),
    author_id        INTEGER NOT NULL REFERENCES users(id),
    title            TEXT    NOT NULL,
    created_at       INTEGER NOT NULL,
    last_activity_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS posts (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    thread_id  INTEGER NOT NULL REFERENCES threads(id),
    author_id  INTEGER NOT NULL REFERENCES users(id),
    body       TEXT    NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_threads_board_last ON threads(board_id, last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_author   ON threads(author_id);
CREATE INDEX IF NOT EXISTS idx_posts_thread     ON posts(thread_id, id);
CREATE INDEX IF NOT EXISTS idx_posts_author     ON posts(author_id);

INSERT OR IGNORE INTO boards (slug, name) VALUES ('main', 'Main Board');
`

func migrate(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}
	if v < 1 {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration: %w", err)
		}
		if _, err := tx.Exec(schemaV1); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply schema v1: %w", err)
		}
		if _, err := tx.Exec("PRAGMA user_version = 1"); err != nil {
			tx.Rollback()
			return fmt.Errorf("set user_version: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration: %w", err)
		}
	}
	return nil
}