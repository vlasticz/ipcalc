package storage

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Saved is a persisted calculation. input_json holds the inputs only; the
// result is recomputed on view so saved rows stay forward-compatible as
// calc output evolves.
type Saved struct {
	Slug      string
	Label     string
	Kind      string // "single" | "split-equal" | "split-vlsm"
	InputJSON string
	ExpiresAt *time.Time
	CreatedAt time.Time
	ViewedAt  time.Time
}

var ErrNotFound = errors.New("saved: not found")

// Create inserts a new row with a freshly-generated slug. Caller supplies
// the input JSON; storage doesn't introspect it. expiresAt nil means
// no expiry.
//
// TODO(v1): wire into POST /saved once the calc UI has a save button.
func (s *Storage) CreateSaved(label, kind, inputJSON string, expiresAt *time.Time) (*Saved, error) {
	slug, err := newSlug()
	if err != nil {
		return nil, fmt.Errorf("new slug: %w", err)
	}
	now := time.Now().UTC()
	var exp sql.NullInt64
	if expiresAt != nil {
		exp = sql.NullInt64{Int64: expiresAt.Unix(), Valid: true}
	}
	_, err = s.db.Exec(`
		INSERT INTO saved (slug, label, kind, input_json, expires_at, created_at, viewed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, slug, label, kind, inputJSON, exp, now.Unix(), now.Unix())
	if err != nil {
		return nil, fmt.Errorf("insert: %w", err)
	}
	return &Saved{
		Slug: slug, Label: label, Kind: kind, InputJSON: inputJSON,
		ExpiresAt: expiresAt, CreatedAt: now, ViewedAt: now,
	}, nil
}

// GetSaved loads a row by slug. Also touches viewed_at.
func (s *Storage) GetSaved(slug string) (*Saved, error) {
	row := s.db.QueryRow(`
		SELECT slug, label, kind, input_json, expires_at, created_at, viewed_at
		FROM saved WHERE slug = ?
	`, slug)
	out := &Saved{}
	var exp sql.NullInt64
	var created, viewed int64
	if err := row.Scan(&out.Slug, &out.Label, &out.Kind, &out.InputJSON, &exp, &created, &viewed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	if exp.Valid {
		t := time.Unix(exp.Int64, 0).UTC()
		out.ExpiresAt = &t
	}
	out.CreatedAt = time.Unix(created, 0).UTC()
	out.ViewedAt = time.Unix(viewed, 0).UTC()

	// Touch viewed_at; best-effort, ignore errors.
	_, _ = s.db.Exec(`UPDATE saved SET viewed_at = ? WHERE slug = ?`, time.Now().UTC().Unix(), slug)
	return out, nil
}

// ListSaved returns every saved row, newest first. Filters out expired rows.
func (s *Storage) ListSaved() ([]*Saved, error) {
	now := time.Now().UTC().Unix()
	rows, err := s.db.Query(`
		SELECT slug, label, kind, input_json, expires_at, created_at, viewed_at
		FROM saved
		WHERE expires_at IS NULL OR expires_at > ?
		ORDER BY created_at DESC
	`, now)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var out []*Saved
	for rows.Next() {
		r := &Saved{}
		var exp sql.NullInt64
		var created, viewed int64
		if err := rows.Scan(&r.Slug, &r.Label, &r.Kind, &r.InputJSON, &exp, &created, &viewed); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if exp.Valid {
			t := time.Unix(exp.Int64, 0).UTC()
			r.ExpiresAt = &t
		}
		r.CreatedAt = time.Unix(created, 0).UTC()
		r.ViewedAt = time.Unix(viewed, 0).UTC()
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Storage) DeleteSaved(slug string) error {
	res, err := s.db.Exec(`DELETE FROM saved WHERE slug = ?`, slug)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// newSlug generates an 8-char URL-safe random ID (~40 bits of entropy,
// plenty for a homelab scope and not enumerable).
func newSlug() (string, error) {
	var b [5]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	return strings.ToLower(enc), nil
}
