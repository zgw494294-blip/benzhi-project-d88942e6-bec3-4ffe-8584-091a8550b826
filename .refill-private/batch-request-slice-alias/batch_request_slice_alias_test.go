package batchrequests_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"termpack/internal/application"
	"termpack/internal/domain"
	"termpack/internal/httpapi"
)

type gatedStore struct {
	repo    application.Repository
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (s *gatedStore) InTransaction(ctx context.Context, fn func(application.Repository) error) error {
	if s.calls.Add(1) == 1 {
		close(s.entered)
		<-s.release
	}
	return fn(s.repo)
}

func (s *gatedStore) View(ctx context.Context, fn func(application.Repository) error) error {
	return fn(s.repo)
}

func (s *gatedStore) Close() error { return nil }

type memoryRepository struct {
	packs   map[string]domain.TermPack
	entries map[string][]domain.TermEntry
}

func (r *memoryRepository) InsertPack(_ context.Context, pack domain.TermPack) error {
	r.packs[pack.ID] = pack
	return nil
}

func (r *memoryRepository) GetPack(_ context.Context, id string) (domain.TermPack, error) {
	pack, ok := r.packs[id]
	if !ok {
		return domain.TermPack{}, domain.ErrNotFound
	}
	return pack, nil
}

func (r *memoryRepository) ListPacks(context.Context) ([]domain.TermPack, error) { return nil, nil }

func (r *memoryRepository) UpdatePack(_ context.Context, pack domain.TermPack, expected uint64) error {
	if r.packs[pack.ID].Version != expected {
		return domain.ErrVersionConflict
	}
	r.packs[pack.ID] = pack
	return nil
}

func (r *memoryRepository) UpdatePackMetadata(context.Context, domain.TermPack, uint64) error {
	return nil
}

func (r *memoryRepository) InsertEntry(_ context.Context, entry domain.TermEntry) error {
	r.entries[entry.TermPackID] = append(r.entries[entry.TermPackID], entry)
	return nil
}

func (r *memoryRepository) UpdateEntry(context.Context, domain.TermEntry, bool) error { return nil }

func (r *memoryRepository) EntriesForRevision(_ context.Context, packID string, revision int) ([]domain.TermEntry, error) {
	var result []domain.TermEntry
	for _, entry := range r.entries[packID] {
		if entry.Revision == revision {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (r *memoryRepository) AllEntries(_ context.Context, packID string) ([]domain.TermEntry, error) {
	return append([]domain.TermEntry(nil), r.entries[packID]...), nil
}

func (r *memoryRepository) GetEntry(context.Context, string, string, int) (domain.TermEntry, error) {
	return domain.TermEntry{}, domain.ErrNotFound
}

func (r *memoryRepository) InsertFinding(context.Context, domain.RehearsalFinding) error { return nil }

func (r *memoryRepository) UpdateFinding(context.Context, domain.RehearsalFinding) error { return nil }

func (r *memoryRepository) Findings(context.Context, string) ([]domain.RehearsalFinding, error) {
	return nil, nil
}

func (r *memoryRepository) GetFinding(context.Context, string, string) (domain.RehearsalFinding, error) {
	return domain.RehearsalFinding{}, domain.ErrNotFound
}

func (r *memoryRepository) InsertCertificate(context.Context, domain.ReleaseCertificate) error {
	return nil
}

func (r *memoryRepository) Certificate(context.Context, string) (*domain.ReleaseCertificate, error) {
	return nil, nil
}

func (r *memoryRepository) GetCommandResult(context.Context, string) (string, json.RawMessage, bool, error) {
	return "", nil, false, nil
}

func (r *memoryRepository) PutCommandResult(context.Context, string, string, json.RawMessage) error {
	return nil
}

func (r *memoryRepository) AppendAudit(context.Context, application.AuditRecord) error { return nil }

func (r *memoryRepository) AuditRecords(context.Context, string) ([]application.AuditRecord, error) {
	return nil, nil
}

func TestConcurrentBatchRequestsOwnDecodedEntries(t *testing.T) {
	now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	packA, err := domain.NewTermPack("pack-a", "A 会场", "中文", "英语", now)
	if err != nil {
		t.Fatal(err)
	}
	packB, err := domain.NewTermPack("pack-b", "B 会场", "中文", "英语", now)
	if err != nil {
		t.Fatal(err)
	}
	repo := &memoryRepository{
		packs:   map[string]domain.TermPack{packA.ID: packA, packB.ID: packB},
		entries: map[string][]domain.TermEntry{},
	}

	gate := &gatedStore{repo: repo, entered: make(chan struct{}), release: make(chan struct{})}
	service := application.NewService(gate)
	mux := http.NewServeMux()
	httpapi.New(service).Register(mux)

	requestA := httptest.NewRequest(http.MethodPost, "/api/v1/term-packs/"+packA.ID+"/entries/batch", strings.NewReader(batchBody("A 术语", "term A", packA.Version, "batch-a")))
	requestA.Header.Set("Content-Type", "application/json")
	responseA := httptest.NewRecorder()
	doneA := make(chan struct{})
	go func() {
		mux.ServeHTTP(responseA, requestA)
		close(doneA)
	}()

	<-gate.entered
	requestB := httptest.NewRequest(http.MethodPost, "/api/v1/term-packs/"+packB.ID+"/entries/batch", strings.NewReader(batchBody("B 术语", "term B", packB.Version, "batch-b")))
	requestB.Header.Set("Content-Type", "application/json")
	responseB := httptest.NewRecorder()
	mux.ServeHTTP(responseB, requestB)
	if responseB.Code != http.StatusCreated {
		t.Fatalf("B 请求应成功，实际状态码 %d，响应 %s", responseB.Code, responseB.Body.String())
	}

	close(gate.release)
	<-doneA
	if responseA.Code != http.StatusCreated {
		t.Fatalf("A 请求应成功，实际状态码 %d，响应 %s", responseA.Code, responseA.Body.String())
	}

	storedA, err := service.Get(context.Background(), packA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(storedA.Entries) != 1 || storedA.Entries[0].SourceTerm != "A 术语" || storedA.Entries[0].PreferredTranslation != "term A" {
		t.Fatalf("A 请求的词条被另一并发请求污染：%+v", storedA.Entries)
	}
}

func batchBody(sourceTerm, translation string, version uint64, key string) string {
	return `{"entries":[{"sourceTerm":"` + sourceTerm + `","preferredTranslation":"` + translation + `","definition":"定义","context":"语境","evidence":"依据"}],"expectedVersion":` + strconv.FormatUint(version, 10) + `,"idempotencyKey":"` + key + `"}`
}
