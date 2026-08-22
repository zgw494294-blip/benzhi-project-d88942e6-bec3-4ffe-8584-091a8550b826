package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"termpack/internal/application"
	"termpack/internal/domain"
)

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type repository struct{ exec executor }

func timeText(t time.Time) string           { return t.UTC().Format(time.RFC3339Nano) }
func parseTime(v string) (time.Time, error) { return time.Parse(time.RFC3339Nano, v) }

func (r *repository) InsertPack(ctx context.Context, p domain.TermPack) error {
	_, err := r.exec.ExecContext(ctx, `INSERT INTO term_packs(id,conference_name,source_language,target_language,status,current_revision,frozen_revision,version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, p.ID, p.ConferenceName, p.SourceLanguage, p.TargetLanguage, p.Status, p.CurrentRevision, p.FrozenRevision, p.Version, timeText(p.CreatedAt), timeText(p.UpdatedAt))
	return err
}

func scanPack(row interface{ Scan(...any) error }) (domain.TermPack, error) {
	var p domain.TermPack
	var created, updated string
	err := row.Scan(&p.ID, &p.ConferenceName, &p.SourceLanguage, &p.TargetLanguage, &p.Status, &p.CurrentRevision, &p.FrozenRevision, &p.Version, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return p, domain.ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.CreatedAt, err = parseTime(created)
	if err != nil {
		return p, err
	}
	p.UpdatedAt, err = parseTime(updated)
	return p, err
}

func (r *repository) GetPack(ctx context.Context, id string) (domain.TermPack, error) {
	return scanPack(r.exec.QueryRowContext(ctx, `SELECT id,conference_name,source_language,target_language,status,current_revision,frozen_revision,version,created_at,updated_at FROM term_packs WHERE id=?`, id))
}

func (r *repository) ListPacks(ctx context.Context) ([]domain.TermPack, error) {
	rows, err := r.exec.QueryContext(ctx, `SELECT id,conference_name,source_language,target_language,status,current_revision,frozen_revision,version,created_at,updated_at FROM term_packs ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var packs []domain.TermPack
	for rows.Next() {
		p, err := scanPack(rows)
		if err != nil {
			return nil, err
		}
		packs = append(packs, p)
	}
	return packs, rows.Err()
}

func (r *repository) UpdatePack(ctx context.Context, p domain.TermPack, expected uint64) error {
	result, err := r.exec.ExecContext(ctx, `UPDATE term_packs SET status=?,current_revision=?,frozen_revision=?,version=?,updated_at=? WHERE id=? AND version=?`, p.Status, p.CurrentRevision, p.FrozenRevision, p.Version, timeText(p.UpdatedAt), p.ID, expected)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return domain.ErrVersionConflict
	}
	return nil
}

func (r *repository) UpdatePackMetadata(ctx context.Context, p domain.TermPack, expected uint64) error {
	result, err := r.exec.ExecContext(ctx, `UPDATE term_packs SET conference_name=?,source_language=?,target_language=?,version=?,updated_at=? WHERE id=? AND version=?`, p.ConferenceName, p.SourceLanguage, p.TargetLanguage, p.Version, timeText(p.UpdatedAt), p.ID, expected)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return domain.ErrVersionConflict
	}
	return nil
}

func (r *repository) GetCommandResult(ctx context.Context, key string) (string, json.RawMessage, bool, error) {
	var action string
	var raw []byte
	err := r.exec.QueryRowContext(ctx, `SELECT action,result_json FROM command_results WHERE idempotency_key=?`, key).Scan(&action, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, false, nil
	}
	return action, raw, err == nil, err
}

func (r *repository) PutCommandResult(ctx context.Context, key, action string, raw json.RawMessage) error {
	_, err := r.exec.ExecContext(ctx, `INSERT INTO command_results(idempotency_key,action,result_json) VALUES(?,?,?)`, key, action, []byte(raw))
	return err
}

func (r *repository) AppendAudit(ctx context.Context, a application.AuditRecord) error {
	_, err := r.exec.ExecContext(ctx, `INSERT INTO audit_records(id,term_pack_id,action,from_status,to_status,revision,payload,created_at) VALUES(?,?,?,?,?,?,?,?)`, a.ID, a.TermPackID, a.Action, a.FromStatus, a.ToStatus, a.Revision, []byte(a.Payload), timeText(a.CreatedAt))
	if err != nil {
		return fmt.Errorf("写入审计记录: %w", err)
	}
	return nil
}

func (r *repository) AuditRecords(ctx context.Context, packID string) ([]application.AuditRecord, error) {
	rows, err := r.exec.QueryContext(ctx, `SELECT id,term_pack_id,action,from_status,to_status,revision,payload,created_at FROM audit_records WHERE term_pack_id=? ORDER BY created_at DESC,id DESC`, packID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []application.AuditRecord{}
	for rows.Next() {
		var record application.AuditRecord
		var payload []byte
		var created string
		if err := rows.Scan(&record.ID, &record.TermPackID, &record.Action, &record.FromStatus, &record.ToStatus, &record.Revision, &payload, &created); err != nil {
			return nil, err
		}
		record.Payload = payload
		record.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}
