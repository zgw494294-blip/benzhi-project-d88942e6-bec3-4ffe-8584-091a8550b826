package contextcancelreport

import (
	"context"
	"errors"
	"testing"

	"termpack/internal/application"
	"termpack/internal/store/sqlite"
)

func TestCanceledFindingReportStopsBeforeReadingStore(t *testing.T) {
	store, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.NewService(store)
	view, err := service.Create(context.Background(), application.CreatePack{
		ConferenceName: "取消传播测试会议",
		SourceLanguage: "中文",
		TargetLanguage: "英语",
		IdempotencyKey: "context-create",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.FindingsReport(ctx, application.FindingFilter{PackID: view.Pack.ID})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FindingsReport 应传播已取消 context，实际错误=%v", err)
	}
}
