// Package storage owns SQLite — schema migration on startup, plus typed
// methods on Storage for each persisted entity. modernc.org/sqlite is used
// to keep the build CGO-free.
package storage

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Storage is the typed wrapper around *sql.DB.
type Storage struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS saved (
  slug        TEXT PRIMARY KEY,
  label       TEXT NOT NULL DEFAULT '',
  kind        TEXT NOT NULL,
  input_json  TEXT NOT NULL,
  expires_at  INTEGER,
  created_at  INTEGER NOT NULL,
  viewed_at   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS saved_created_idx ON saved(created_at DESC);
`

// Open opens (or creates) the SQLite database at path. The containing
// directory is created if missing. WAL + a busy timeout are set via query
// params so concurrent readers don't trip over a writer.
func Open(path string) (*Storage, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=%s&_pragma=%s&_pragma=%s",
		url.PathEscape(path),
		url.QueryEscape("journal_mode(WAL)"),
		url.QueryEscape("busy_timeout(5000)"),
		url.QueryEscape("foreign_keys(ON)"),
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
