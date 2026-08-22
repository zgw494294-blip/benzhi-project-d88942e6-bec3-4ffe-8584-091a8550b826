package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"termpack/internal/domain"
)

func (r *repository) InsertFinding(ctx context.Context, f domain.RehearsalFinding) error {
	_, err := r.exec.ExecContext(ctx, `INSERT INTO rehearsal_findings(id,term_pack_id,frozen_revision,entry_id,scenario,severity,observation,resolution,status,reported_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, f.ID, f.TermPackID, f.FrozenRevision, f.EntryID, f.Scenario, f.Severity, f.Observation, f.Resolution, f.Status, timeText(f.ReportedAt))
	return err
}

func (r *repository) UpdateFinding(ctx context.Context, f domain.RehearsalFinding) error {
	result, err := r.exec.ExecContext(ctx, `UPDATE rehearsal_findings SET resolution=?,status=? WHERE id=? AND term_pack_id=?`, f.Resolution, f.Status, f.ID, f.TermPackID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return domain.ErrNotFound
	}
	return nil
}

func scanFinding(row interface{ Scan(...any) error }) (domain.RehearsalFinding, error) {
	var f domain.RehearsalFinding
	var reported string
	err := row.Scan(&f.ID, &f.TermPackID, &f.FrozenRevision, &f.EntryID, &f.Scenario, &f.Severity, &f.Observation, &f.Resolution, &f.Status, &reported)
	if errors.Is(err, sql.ErrNoRows) {
		return f, domain.ErrNotFound
	}
	if err != nil {
		return f, err
	}
	f.ReportedAt, err = parseTime(reported)
	return f, err
}

const selectFindings = `SELECT id,term_pack_id,frozen_revision,entry_id,scenario,severity,observation,resolution,status,reported_at FROM rehearsal_findings`

func (r *repository) Findings(ctx context.Context, packID string) ([]domain.RehearsalFinding, error) {
	rows, err := r.exec.QueryContext(ctx, selectFindings+` WHERE term_pack_id=? ORDER BY reported_at DESC`, packID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	findings := []domain.RehearsalFinding{}
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		findings = append(findings, f)
	}
	return findings, rows.Err()
}

func (r *repository) GetFinding(ctx context.Context, packID, id string) (domain.RehearsalFinding, error) {
	return scanFinding(r.exec.QueryRowContext(ctx, selectFindings+` WHERE term_pack_id=? AND id=?`, packID, id))
}
