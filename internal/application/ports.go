package application

import (
	"context"
	"encoding/json"

	"termpack/internal/domain"
)

type Repository interface {
	InsertPack(context.Context, domain.TermPack) error
	GetPack(context.Context, string) (domain.TermPack, error)
	ListPacks(context.Context) ([]domain.TermPack, error)
	UpdatePack(context.Context, domain.TermPack, uint64) error
	UpdatePackMetadata(context.Context, domain.TermPack, uint64) error
	InsertEntry(context.Context, domain.TermEntry) error
	UpdateEntry(context.Context, domain.TermEntry, bool) error
	EntriesForRevision(context.Context, string, int) ([]domain.TermEntry, error)
	AllEntries(context.Context, string) ([]domain.TermEntry, error)
	GetEntry(context.Context, string, string, int) (domain.TermEntry, error)
	InsertFinding(context.Context, domain.RehearsalFinding) error
	UpdateFinding(context.Context, domain.RehearsalFinding) error
	Findings(context.Context, string) ([]domain.RehearsalFinding, error)
	GetFinding(context.Context, string, string) (domain.RehearsalFinding, error)
	InsertCertificate(context.Context, domain.ReleaseCertificate) error
	Certificate(context.Context, string) (*domain.ReleaseCertificate, error)
	GetCommandResult(context.Context, string) (string, json.RawMessage, bool, error)
	PutCommandResult(context.Context, string, string, json.RawMessage) error
	AppendAudit(context.Context, AuditRecord) error
	AuditRecords(context.Context, string) ([]AuditRecord, error)
}

type Store interface {
	InTransaction(context.Context, func(Repository) error) error
	View(context.Context, func(Repository) error) error
	Close() error
}
