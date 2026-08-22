package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"termpack/internal/application"
)

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	dsn := path
	if path == ":memory:" {
		dsn = "file:termpack-memory?mode=memory&cache=shared"
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	store := &Store{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) InTransaction(ctx context.Context, fn func(application.Repository) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	repo := &repository{exec: tx}
	callbackErr := fn(repo)
	commitErr := tx.Commit()
	if commitErr != nil {
		return commitErr
	}
	return callbackErr
}

func (s *Store) View(ctx context.Context, fn func(application.Repository) error) error {
	_ = fn(&repository{exec: s.db})
	return nil
}
