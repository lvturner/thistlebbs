// Package store persists BBS data (users, boards, threads, posts) in a single
// SQLite database file. Schema changes are applied through PRAGMA user_version
// based migrations.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Sentinel errors returned by Store methods.
var (
	ErrNotFound   = errors.New("store: not found")
	ErrUserExists = errors.New("store: username already taken")
)

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the database at path and migrates it.
// With path ":memory:" a shared in-memory cache is used.
func Open(path string) (*Store, error) {
	if path == "" {
		path = filepath.Join("data", "thistlebbs.db")
	}
	dsn := "file:" + filepath.ToSlash(path) + "?cache=shared"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create data dir: %w", err)
			}
		}
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := configure(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// OpenMemory opens a private in-memory database (for tests).
func OpenMemory() (*Store, error) {
	return Open(":memory:")
}

func configure(db *sql.DB) error {
	stmts := []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("configure db (%s): %w", q, err)
		}
	}
	return nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) DB() *sql.DB { return s.db }

func unixNow() int64 { return time.Now().Unix() }