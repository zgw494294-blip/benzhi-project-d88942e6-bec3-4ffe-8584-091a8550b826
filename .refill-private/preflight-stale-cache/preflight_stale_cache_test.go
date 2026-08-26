package preflightstalecache_test

import (
	"context"
	"testing"

	"termpack/internal/application"
	"termpack/internal/store/sqlite"
)

func TestPreflightRefreshesAfterPackMutation(t *testing.T) {
	store, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	service := application.NewService(store)
	ctx := context.Background()
	view, err := service.Create(ctx, application.CreatePack{
		ConferenceName: "缓存一致性测试会",
		SourceLanguage: "中文",
		TargetLanguage: "英语",
		IdempotencyKey: "preflight-cache-create",
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.AddEntry(ctx, application.AddEntry{
		PackID: view.Pack.ID, SourceTerm: "同声传译", PreferredTranslation: "simultaneous interpreting",
		Definition: "讲话与口译近乎同步进行", Context: "主旨演讲", Evidence: "会议议程",
		ExpectedVersion: view.Pack.Version, IdempotencyKey: "preflight-cache-add-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	first, err := service.Preflight(ctx, view.Pack.ID, view.Pack.CurrentRevision)
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 1 {
		t.Fatalf("首次预检应读取 1 条词条，实际 %d", first.Total)
	}

	view, err = service.AddEntry(ctx, application.AddEntry{
		PackID: view.Pack.ID, SourceTerm: "交替传译", PreferredTranslation: "consecutive interpreting",
		Definition: "讲话与口译交替进行", Context: "圆桌讨论", Evidence: "术语准备材料",
		ExpectedVersion: view.Pack.Version, IdempotencyKey: "preflight-cache-add-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Preflight(ctx, view.Pack.ID, view.Pack.CurrentRevision)
	if err != nil {
		t.Fatal(err)
	}
	if second.Total != 2 || !second.CanSubmit {
		t.Fatalf("词条新增后预检应重新读取 store：total=%d canSubmit=%v", second.Total, second.CanSubmit)
	}
}
