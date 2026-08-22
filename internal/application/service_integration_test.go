package application_test

import (
	"context"
	"testing"

	"termpack/internal/application"
	"termpack/internal/domain"
	"termpack/internal/store/sqlite"
)

func TestCorrectionWorkflowPersistsHistoryAndCertificate(t *testing.T) {
	store, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.NewService(store)
	ctx := context.Background()
	view, err := service.Create(ctx, application.CreatePack{ConferenceName: "专业口译峰会", SourceLanguage: "中文", TargetLanguage: "英语", IdempotencyKey: "create"})
	if err != nil {
		t.Fatal(err)
	}
	packID := view.Pack.ID
	view, err = service.AddEntry(ctx, application.AddEntry{PackID: packID, SourceTerm: "交替传译", PreferredTranslation: "consecutive interpreting", Definition: "发言与口译交替进行", Context: "工作坊", Evidence: "会议议程", ExpectedVersion: view.Pack.Version, IdempotencyKey: "add"})
	if err != nil {
		t.Fatal(err)
	}
	entryID := view.Entries[0].ID
	view, err = service.Submit(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "submit-1"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.ReviewEntry(ctx, application.ReviewEntry{PackID: packID, EntryID: entryID, Decision: domain.DecisionAccepted, ExpectedVersion: view.Pack.Version, IdempotencyKey: "review-1"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.Freeze(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "freeze-1"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.StartRehearsal(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "rehearsal-1"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.AddFinding(ctx, application.AddFinding{PackID: packID, EntryID: entryID, Scenario: "问答环节", Severity: domain.SeverityMajor, Observation: "译法需要明确模式", ExpectedVersion: view.Pack.Version, IdempotencyKey: "finding"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.ResolveFinding(ctx, application.ResolveFinding{PackID: packID, FindingID: view.Findings[0].ID, Resolution: "补充定义", ExpectedVersion: view.Pack.Version, IdempotencyKey: "resolve"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.BeginRevision(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "revision"})
	if err != nil {
		t.Fatal(err)
	}
	newEntry := view.Entries[0]
	view, err = service.UpdateEntry(ctx, application.UpdateEntry{PackID: packID, EntryID: newEntry.ID, SourceTerm: newEntry.SourceTerm, PreferredTranslation: newEntry.PreferredTranslation, Definition: "发言结束后由口译员完整传译", Context: newEntry.Context, Evidence: newEntry.Evidence, ExpectedVersion: view.Pack.Version, IdempotencyKey: "edit-2"})
	if err != nil {
		t.Fatal(err)
	}
	view, _ = service.Submit(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "submit-2"})
	view, _ = service.ReviewEntry(ctx, application.ReviewEntry{PackID: packID, EntryID: newEntry.ID, Decision: domain.DecisionAccepted, ExpectedVersion: view.Pack.Version, IdempotencyKey: "review-2"})
	view, _ = service.Freeze(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "freeze-2"})
	view, _ = service.StartRehearsal(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "rehearsal-2"})
	view, err = service.Release(ctx, application.Release{PackID: packID, ApprovedBy: "发布负责人", ExpectedVersion: view.Pack.Version, IdempotencyKey: "release"})
	if err != nil {
		t.Fatal(err)
	}
	if view.Pack.Status != domain.StatusReleased || view.Certificate == nil {
		t.Fatal("完整纠错流程必须生成发布凭据")
	}
	if len(view.EntryHistory) != 2 || len(view.AuditTrail) < 12 {
		t.Fatalf("应保留两版词条与完整审计，词条=%d 审计=%d", len(view.EntryHistory), len(view.AuditTrail))
	}
	reused, err := service.Release(ctx, application.Release{PackID: packID, ApprovedBy: "不同输入也不重复执行", ExpectedVersion: 1, IdempotencyKey: "release"})
	if err != nil || reused.Certificate.ID != view.Certificate.ID {
		t.Fatal("相同幂等键必须复用原发布结果")
	}
}

func TestExpectedVersionConflictRollsBack(t *testing.T) {
	store, _ := sqlite.Open(":memory:")
	defer store.Close()
	service := application.NewService(store)
	view, _ := service.Create(context.Background(), application.CreatePack{ConferenceName: "测试会议", SourceLanguage: "中", TargetLanguage: "英", IdempotencyKey: "create-conflict"})
	_, err := service.AddEntry(context.Background(), application.AddEntry{PackID: view.Pack.ID, SourceTerm: "术语", PreferredTranslation: "term", Definition: "定义", Context: "语境", Evidence: "依据", ExpectedVersion: view.Pack.Version + 1, IdempotencyKey: "conflict"})
	if domain.ErrorCode(err) != "version_conflict" {
		t.Fatalf("期望 version_conflict，实际 %v", err)
	}
	latest, _ := service.Get(context.Background(), view.Pack.ID)
	if len(latest.Entries) != 0 || latest.Pack.Version != view.Pack.Version {
		t.Fatal("版本冲突事务不得产生部分写入")
	}
}
