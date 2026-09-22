package store

import (
	"database/sql"
	"errors"
	"fmt"

	sqlitemod "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// User is a registered board user.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Location     string
	Bio          string
	JoinedAt     int64 // unix seconds
	LastLogin  int64 // unix seconds, 0 = never logged in
}

// CreateUser inserts a new user. PasswordHash must already be hashed by the
// caller. Returns ErrUserExists on a username collision.
func (s *Store) CreateUser(u *User) error {
	res, err := s.db.Exec(`
		INSERT INTO users (username, password_hash, location, bio, joined_at)
		VALUES (?, ?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.Location, u.Bio, u.JoinedAt)
	if err != nil {
	var sqliteErr *sqlitemod.Error
	if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return ErrUserExists
	}
		return fmt.Errorf("create user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("create user id: %w", err)
	}
	u.ID = id
	return nil
}

// UserByUsername looks a user up by case-insensitive username.
func (s *Store) UserByUsername(username string) (*User, error) {
	return scanUser(s.db.QueryRow(
		`SELECT id, username, password_hash, location, bio, joined_at, last_login
		   FROM users WHERE username = ?`, username))
}

// UserByID looks a user up by primary key.
func (s *Store) UserByID(id int64) (*User, error) {
	return scanUser(s.db.QueryRow(
		`SELECT id, username, password_hash, location, bio, joined_at, last_login
		   FROM users WHERE id = ?`, id))
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash,
		&u.Location, &u.Bio, &u.JoinedAt, &u.LastLogin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}

// SetLastLogin stamps the user's last login time (unix seconds).
func (s *Store) SetLastLogin(id, t int64) error {
	if _, err := s.db.Exec(`UPDATE users SET last_login = ? WHERE id = ?`, t, id); err != nil {
		return fmt.Errorf("set last login: %w", err)
	}
	return nil
}

// AllUsers lists every user, most recent last login first. Users who never
// logged in (last_login 0) sort last, oldest member first among them.
func (s *Store) AllUsers() ([]User, error) {
	rows, err := s.db.Query(`
		SELECT id, username, password_hash, location, bio, joined_at, last_login
		FROM users ORDER BY last_login DESC, joined_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Location, &u.Bio, &u.JoinedAt, &u.LastLogin); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateProfile refreshes a user's editable profile fields.
func (s *Store) UpdateProfile(id int64, location, bio string) error {
	res, err := s.db.Exec(
		`UPDATE users SET location = ?, bio = ? WHERE id = ?`,
		location, bio, id)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// UserStats returns how many threads a user started and how many posts
// (including thread openers) they have written.
func (s *Store) UserStats(id int64) (threads, posts int64, err error) {
	err = s.db.QueryRow(`
		SELECT (SELECT COUNT(*) FROM threads WHERE author_id = ?),
		       (SELECT COUNT(*) FROM posts    WHERE author_id = ?)`, id, id).
		Scan(&threads, &posts)
	return threads, posts, err
}