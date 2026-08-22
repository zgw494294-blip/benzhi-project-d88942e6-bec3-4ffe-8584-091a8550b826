package httpapi

import (
	"termpack/internal/application"
	"termpack/internal/domain"
)

type createPackRequest struct {
	ConferenceName string `json:"conferenceName"`
	SourceLanguage string `json:"sourceLanguage"`
	TargetLanguage string `json:"targetLanguage"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type addEntryRequest struct {
	SourceTerm           string `json:"sourceTerm"`
	PreferredTranslation string `json:"preferredTranslation"`
	Definition           string `json:"definition"`
	Context              string `json:"context"`
	Evidence             string `json:"evidence"`
	ExpectedVersion      uint64 `json:"expectedVersion"`
	IdempotencyKey       string `json:"idempotencyKey"`
}

type batchEntriesRequest struct {
	Entries         []application.EntryInput `json:"entries"`
	ExpectedVersion uint64                   `json:"expectedVersion"`
	IdempotencyKey  string                   `json:"idempotencyKey"`
}

type metadataRequest struct {
	ConferenceName  string `json:"conferenceName"`
	SourceLanguage  string `json:"sourceLanguage"`
	TargetLanguage  string `json:"targetLanguage"`
	ExpectedVersion uint64 `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type updateEntryRequest = addEntryRequest

type versionedRequest struct {
	ExpectedVersion uint64 `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type reviewEntryRequest struct {
	Decision        domain.EntryDecision `json:"decision"`
	Translation     string               `json:"translation"`
	EditorNote      string               `json:"editorNote"`
	ExpectedVersion uint64               `json:"expectedVersion"`
	IdempotencyKey  string               `json:"idempotencyKey"`
}

type batchReviewRequest struct {
	Items           []application.BatchReviewItem `json:"items"`
	ExpectedVersion uint64                        `json:"expectedVersion"`
	IdempotencyKey  string                        `json:"idempotencyKey"`
}

type addFindingRequest struct {
	EntryID         string                 `json:"entryID"`
	Scenario        string                 `json:"scenario"`
	Severity        domain.FindingSeverity `json:"severity"`
	Observation     string                 `json:"observation"`
	ExpectedVersion uint64                 `json:"expectedVersion"`
	IdempotencyKey  string                 `json:"idempotencyKey"`
}

type resolveFindingRequest struct {
	Resolution      string `json:"resolution"`
	ExpectedVersion uint64 `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type batchResolveRequest struct {
	Items           []application.CloseFindingItem `json:"items"`
	ExpectedVersion uint64                         `json:"expectedVersion"`
	IdempotencyKey  string                         `json:"idempotencyKey"`
}

type releaseRequest struct {
	ApprovedBy      string `json:"approvedBy"`
	ExpectedVersion uint64 `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}
