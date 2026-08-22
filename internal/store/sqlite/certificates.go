package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"termpack/internal/domain"
)

func (r *repository) InsertCertificate(ctx context.Context, c domain.ReleaseCertificate) error {
	_, err := r.exec.ExecContext(ctx, `INSERT INTO release_certificates(id,term_pack_id,released_revision,entry_count,approved_by,approved_at,content_digest,snapshot_json) VALUES(?,?,?,?,?,?,?,?)`, c.ID, c.TermPackID, c.ReleasedRevision, c.EntryCount, c.ApprovedBy, timeText(c.ApprovedAt), c.ContentDigest, []byte(c.SnapshotJSON))
	return err
}

func (r *repository) Certificate(ctx context.Context, packID string) (*domain.ReleaseCertificate, error) {
	var c domain.ReleaseCertificate
	var approved string
	var snapshot []byte
	err := r.exec.QueryRowContext(ctx, `SELECT id,term_pack_id,released_revision,entry_count,approved_by,approved_at,content_digest,snapshot_json FROM release_certificates WHERE term_pack_id=?`, packID).Scan(&c.ID, &c.TermPackID, &c.ReleasedRevision, &c.EntryCount, &c.ApprovedBy, &approved, &c.ContentDigest, &snapshot)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.ApprovedAt, err = parseTime(approved)
	c.SnapshotJSON = snapshot
	return &c, err
}
