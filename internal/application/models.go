package application

import (
	"encoding/json"
	"time"

	"termpack/internal/domain"
)

type PackView struct {
	Pack         domain.TermPack            `json:"pack"`
	Entries      []domain.TermEntry         `json:"entries"`
	EntryHistory []domain.TermEntry         `json:"entryHistory"`
	Findings     []domain.RehearsalFinding  `json:"findings"`
	Certificate  *domain.ReleaseCertificate `json:"certificate,omitempty"`
	AuditTrail   []AuditRecord              `json:"auditTrail"`
}

type AuditRecord struct {
	ID         string          `json:"id"`
	TermPackID string          `json:"termPackID"`
	Action     string          `json:"action"`
	FromStatus string          `json:"fromStatus"`
	ToStatus   string          `json:"toStatus"`
	Revision   int             `json:"revision"`
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"createdAt"`
}

type CreatePack struct {
	ConferenceName string
	SourceLanguage string
	TargetLanguage string
	IdempotencyKey string
}

type AddEntry struct {
	PackID, SourceTerm, PreferredTranslation, Definition, Context, Evidence string
	ExpectedVersion                                                         uint64
	IdempotencyKey                                                          string
}

type EntryInput struct {
	SourceTerm           string `json:"sourceTerm"`
	PreferredTranslation string `json:"preferredTranslation"`
	Definition           string `json:"definition"`
	Context              string `json:"context"`
	Evidence             string `json:"evidence"`
}

type BatchAddEntries struct {
	PackID          string
	Entries         []EntryInput
	ExpectedVersion uint64
	IdempotencyKey  string
}

type UpdateMetadata struct {
	PackID          string
	ConferenceName  string
	SourceLanguage  string
	TargetLanguage  string
	ExpectedVersion uint64
	IdempotencyKey  string
}

type UpdateEntry struct {
	PackID, EntryID, SourceTerm, PreferredTranslation, Definition, Context, Evidence string
	ExpectedVersion                                                                  uint64
	IdempotencyKey                                                                   string
}

type VersionedCommand struct {
	PackID          string
	ExpectedVersion uint64
	IdempotencyKey  string
}

type ReviewEntry struct {
	PackID, EntryID, Translation, EditorNote string
	Decision                                 domain.EntryDecision
	ExpectedVersion                          uint64
	IdempotencyKey                           string
}

type BatchReviewItem struct {
	EntryID     string               `json:"entryID"`
	Decision    domain.EntryDecision `json:"decision"`
	Translation string               `json:"translation"`
	EditorNote  string               `json:"editorNote"`
}

type BatchReview struct {
	PackID          string
	Items           []BatchReviewItem
	ExpectedVersion uint64
	IdempotencyKey  string
}

type AddFinding struct {
	PackID, EntryID, Scenario, Observation string
	Severity                               domain.FindingSeverity
	ExpectedVersion                        uint64
	IdempotencyKey                         string
}

type ResolveFinding struct {
	PackID, FindingID, Resolution string
	ExpectedVersion               uint64
	IdempotencyKey                string
}

type FindingFilter struct {
	PackID         string
	FrozenRevision int
	Severity       domain.FindingSeverity
	Status         domain.FindingStatus
}

type FindingResult struct {
	Finding domain.RehearsalFinding `json:"finding"`
	Entry   *domain.TermEntry       `json:"entry,omitempty"`
}

type FindingReport struct {
	TermPackID     string          `json:"termPackID"`
	FrozenRevision int             `json:"frozenRevision"`
	Items          []FindingResult `json:"items"`
	Total          int             `json:"total"`
	Open           int             `json:"open"`
	Closed         int             `json:"closed"`
}

type CloseFindingItem struct {
	FindingID  string `json:"findingID"`
	Resolution string `json:"resolution"`
}

type CloseFindings struct {
	PackID          string
	Items           []CloseFindingItem
	ExpectedVersion uint64
	IdempotencyKey  string
}

type PreflightProblem struct {
	EntryID string `json:"entryID"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PreflightReport struct {
	TermPackID        string             `json:"termPackID"`
	Revision          int                `json:"revision"`
	Problems          []PreflightProblem `json:"problems"`
	Total             int                `json:"total"`
	Pending           int                `json:"pending"`
	Accepted          int                `json:"accepted"`
	Replaced          int                `json:"replaced"`
	Rejected          int                `json:"rejected"`
	CanSubmit         bool               `json:"canSubmit"`
	CanCompleteReview bool               `json:"canCompleteReview"`
}

type DiffItem struct {
	Classification string            `json:"classification"`
	SourceTerm     string            `json:"sourceTerm"`
	Previous       *domain.TermEntry `json:"previous,omitempty"`
	Current        *domain.TermEntry `json:"current,omitempty"`
}

type RevisionDiff struct {
	TermPackID       string     `json:"termPackID"`
	PreviousRevision int        `json:"previousRevision"`
	CurrentRevision  int        `json:"currentRevision"`
	Items            []DiffItem `json:"items"`
}

type CertificateCheck struct {
	Name    string `json:"name"`
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

type CertificateVerification struct {
	TermPackID string             `json:"termPackID"`
	Valid      bool               `json:"valid"`
	Checks     []CertificateCheck `json:"checks"`
}

type Release struct {
	PackID, ApprovedBy string
	ExpectedVersion    uint64
	IdempotencyKey     string
}
