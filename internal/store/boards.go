package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// Board is a discussion board. The current system has a single board.
type Board struct {
	ID   int64
	Slug string
	Name string
}

// BoardBySlug fetches a board by its stable slug.
func (s *Store) BoardBySlug(slug string) (*Board, error) {
	var b Board
	err := s.db.QueryRow(
		`SELECT id, slug, name FROM boards WHERE slug = ?`, slug).
		Scan(&b.ID, &b.Slug, &b.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("board by slug: %w", err)
	}
	return &b, nil
}