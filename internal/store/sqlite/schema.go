package sqlite

import (
	"context"
	"fmt"
)

const schemaVersion = 1

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_versions (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS term_packs (
		id TEXT PRIMARY KEY, conference_name TEXT NOT NULL, source_language TEXT NOT NULL,
		target_language TEXT NOT NULL, status TEXT NOT NULL, current_revision INTEGER NOT NULL,
		frozen_revision INTEGER NOT NULL DEFAULT 0, version INTEGER NOT NULL,
		created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS term_entries (
		id TEXT PRIMARY KEY, term_pack_id TEXT NOT NULL REFERENCES term_packs(id), revision INTEGER NOT NULL,
		source_term TEXT NOT NULL, preferred_translation TEXT NOT NULL, definition TEXT NOT NULL,
		context_text TEXT NOT NULL, evidence TEXT NOT NULL, decision TEXT NOT NULL, editor_note TEXT NOT NULL,
		UNIQUE(term_pack_id, revision, source_term)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_entries_pack_revision ON term_entries(term_pack_id, revision)`,
	`CREATE TABLE IF NOT EXISTS rehearsal_findings (
		id TEXT PRIMARY KEY, term_pack_id TEXT NOT NULL REFERENCES term_packs(id), frozen_revision INTEGER NOT NULL,
		entry_id TEXT NOT NULL, scenario TEXT NOT NULL, severity TEXT NOT NULL, observation TEXT NOT NULL,
		resolution TEXT NOT NULL, status TEXT NOT NULL, reported_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_findings_pack ON rehearsal_findings(term_pack_id, frozen_revision)`,
	`CREATE TABLE IF NOT EXISTS release_certificates (
		id TEXT PRIMARY KEY, term_pack_id TEXT NOT NULL UNIQUE REFERENCES term_packs(id), released_revision INTEGER NOT NULL,
		entry_count INTEGER NOT NULL, approved_by TEXT NOT NULL, approved_at TEXT NOT NULL,
		content_digest TEXT NOT NULL UNIQUE, snapshot_json BLOB NOT NULL,
		UNIQUE(term_pack_id, released_revision)
	)`,
	`CREATE TABLE IF NOT EXISTS audit_records (
		id TEXT PRIMARY KEY, term_pack_id TEXT NOT NULL REFERENCES term_packs(id), action TEXT NOT NULL,
		from_status TEXT NOT NULL, to_status TEXT NOT NULL, revision INTEGER NOT NULL,
		payload BLOB NOT NULL, created_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_audit_pack ON audit_records(term_pack_id, created_at)`,
	`CREATE TABLE IF NOT EXISTS command_results (
		idempotency_key TEXT PRIMARY KEY, action TEXT NOT NULL, result_json BLOB NOT NULL,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
}

func (s *Store) migrate(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, statement := range migrations {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("执行数据库迁移: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO schema_versions(version, applied_at) VALUES (?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))`, schemaVersion); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
