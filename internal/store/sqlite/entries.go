package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"termpack/internal/domain"
)

func (r *repository) InsertEntry(ctx context.Context, e domain.TermEntry) error {
	_, err := r.exec.ExecContext(ctx, `INSERT INTO term_entries(id,term_pack_id,revision,source_term,preferred_translation,definition,context_text,evidence,decision,editor_note) VALUES(?,?,?,?,?,?,?,?,?,?)`, e.ID, e.TermPackID, e.Revision, e.SourceTerm, e.PreferredTranslation, e.Definition, e.Context, e.Evidence, e.Decision, e.EditorNote)
	return err
}

func (r *repository) UpdateEntry(ctx context.Context, e domain.TermEntry, content bool) error {
	query := `UPDATE term_entries SET preferred_translation=?,decision=?,editor_note=? WHERE id=? AND term_pack_id=? AND revision=?`
	args := []any{e.PreferredTranslation, e.Decision, e.EditorNote, e.ID, e.TermPackID, e.Revision}
	if content {
		query = `UPDATE term_entries SET source_term=?,preferred_translation=?,definition=?,context_text=?,evidence=?,decision=?,editor_note=? WHERE id=? AND term_pack_id=? AND revision=?`
		args = []any{e.SourceTerm, e.PreferredTranslation, e.Definition, e.Context, e.Evidence, e.Decision, e.EditorNote, e.ID, e.TermPackID, e.Revision}
	}
	result, err := r.exec.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return domain.ErrNotFound
	}
	return nil
}

func scanEntry(row interface{ Scan(...any) error }) (domain.TermEntry, error) {
	var e domain.TermEntry
	err := row.Scan(&e.ID, &e.TermPackID, &e.Revision, &e.SourceTerm, &e.PreferredTranslation, &e.Definition, &e.Context, &e.Evidence, &e.Decision, &e.EditorNote)
	if errors.Is(err, sql.ErrNoRows) {
		return e, domain.ErrNotFound
	}
	return e, err
}

func (r *repository) entryQuery(ctx context.Context, query string, args ...any) ([]domain.TermEntry, error) {
	rows, err := r.exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []domain.TermEntry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

const selectEntries = `SELECT id,term_pack_id,revision,source_term,preferred_translation,definition,context_text,evidence,decision,editor_note FROM term_entries`

func (r *repository) EntriesForRevision(ctx context.Context, packID string, revision int) ([]domain.TermEntry, error) {
	return r.entryQuery(ctx, selectEntries+` WHERE term_pack_id=? AND revision=? ORDER BY source_term,id`, packID, revision)
}

func (r *repository) AllEntries(ctx context.Context, packID string) ([]domain.TermEntry, error) {
	return r.entryQuery(ctx, selectEntries+` WHERE term_pack_id=? ORDER BY revision DESC,source_term,id`, packID)
}

func (r *repository) GetEntry(ctx context.Context, packID, id string, revision int) (domain.TermEntry, error) {
	return scanEntry(r.exec.QueryRowContext(ctx, selectEntries+` WHERE term_pack_id=? AND id=? AND revision=?`, packID, id, revision))
}
