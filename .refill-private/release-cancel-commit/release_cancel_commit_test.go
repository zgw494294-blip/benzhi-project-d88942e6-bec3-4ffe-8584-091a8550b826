package release_cancel_commit_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"termpack/internal/application"
	"termpack/internal/domain"
	"termpack/internal/httpapi"
	"termpack/internal/store/sqlite"
)

func TestCanceledReleaseCannotCommitCertificate(t *testing.T) {
	store, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	service := application.NewService(store)
	view := prepareRehearsalPack(t, service)

	mux := http.NewServeMux()
	httpapi.New(service).Register(mux)
	body := fmt.Sprintf(`{"approvedBy":"发布负责人","expectedVersion":%d,"idempotencyKey":"release-canceled"}`, view.Pack.Version)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/term-packs/"+view.Pack.ID+"/release", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	canceled, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(canceled)

	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)

	latest, err := service.Get(context.Background(), view.Pack.ID)
	if err != nil {
		t.Fatal(err)
	}
	if response.Code == http.StatusOK || latest.Pack.Status == domain.StatusReleased || latest.Certificate != nil {
		t.Fatalf("已取消的发布请求仍提交事务: status=%d packStatus=%s certificate=%v", response.Code, latest.Pack.Status, latest.Certificate != nil)
	}
}

func prepareRehearsalPack(t *testing.T, service *application.Service) application.PackView {
	t.Helper()
	ctx := context.Background()
	view, err := service.Create(ctx, application.CreatePack{
		ConferenceName: "取消发布测试会议",
		SourceLanguage: "中文",
		TargetLanguage: "英语",
		IdempotencyKey: "release-cancel-create",
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.AddEntry(ctx, application.AddEntry{
		PackID: view.Pack.ID, SourceTerm: "同声传译", PreferredTranslation: "simultaneous interpreting",
		Definition: "口译员同步传达发言内容", Context: "主会场", Evidence: "会议议程",
		ExpectedVersion: view.Pack.Version, IdempotencyKey: "release-cancel-entry",
	})
	if err != nil {
		t.Fatal(err)
	}
	entryID := view.Entries[0].ID
	view, err = service.Submit(ctx, application.VersionedCommand{PackID: view.Pack.ID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "release-cancel-submit"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.ReviewEntry(ctx, application.ReviewEntry{
		PackID: view.Pack.ID, EntryID: entryID, Decision: domain.DecisionAccepted,
		ExpectedVersion: view.Pack.Version, IdempotencyKey: "release-cancel-review",
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.Freeze(ctx, application.VersionedCommand{PackID: view.Pack.ID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "release-cancel-freeze"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.StartRehearsal(ctx, application.VersionedCommand{PackID: view.Pack.ID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "release-cancel-rehearsal"})
	if err != nil {
		t.Fatal(err)
	}
	return view
}
