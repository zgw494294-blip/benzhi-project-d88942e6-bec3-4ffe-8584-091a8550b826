package finding_report_slice_alias_test

import (
	"context"
	"testing"

	"termpack/internal/application"
	"termpack/internal/domain"
	"termpack/internal/store/sqlite"
)

func TestFindingReportsOwnFilteredItems(t *testing.T) {
	store, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.NewService(store)
	ctx := context.Background()
	view, err := service.Create(ctx, application.CreatePack{ConferenceName: "报告别名会议", SourceLanguage: "中文", TargetLanguage: "英语", IdempotencyKey: "create-report-alias"})
	if err != nil {
		t.Fatal(err)
	}
	packID := view.Pack.ID
	view, err = service.AddEntry(ctx, application.AddEntry{PackID: packID, SourceTerm: "术语", PreferredTranslation: "term", Definition: "定义", Context: "语境", Evidence: "依据", ExpectedVersion: view.Pack.Version, IdempotencyKey: "add-report-alias"})
	if err != nil {
		t.Fatal(err)
	}
	entryID := view.Entries[0].ID
	view, err = service.Submit(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "submit-report-alias"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.ReviewEntry(ctx, application.ReviewEntry{PackID: packID, EntryID: entryID, Decision: domain.DecisionAccepted, ExpectedVersion: view.Pack.Version, IdempotencyKey: "review-report-alias"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.Freeze(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "freeze-report-alias"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.StartRehearsal(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "start-report-alias"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.AddFinding(ctx, application.AddFinding{PackID: packID, EntryID: entryID, Scenario: "高语速", Severity: domain.SeverityMajor, Observation: "首译不清", ExpectedVersion: view.Pack.Version, IdempotencyKey: "major-report-alias"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddFinding(ctx, application.AddFinding{PackID: packID, EntryID: entryID, Scenario: "问答", Severity: domain.SeverityMinor, Observation: "术语可再确认", ExpectedVersion: view.Pack.Version, IdempotencyKey: "minor-report-alias"})
	if err != nil {
		t.Fatal(err)
	}
	major, err := service.FindingsReport(ctx, application.FindingFilter{PackID: packID, Severity: domain.SeverityMajor})
	if err != nil {
		t.Fatal(err)
	}
	if len(major.Items) != 1 || major.Items[0].Finding.Severity != domain.SeverityMajor {
		t.Fatalf("第一次筛选应保留 Major 发现，实际 %#v", major.Items)
	}
	majorID := major.Items[0].Finding.ID
	if _, err := service.FindingsReport(ctx, application.FindingFilter{PackID: packID, Severity: domain.SeverityMinor}); err != nil {
		t.Fatal(err)
	}
	if len(major.Items) != 1 || major.Items[0].Finding.ID != majorID || major.Items[0].Finding.Severity != domain.SeverityMajor {
		t.Fatalf("后续筛选不应改写已返回报告，实际 %#v", major.Items)
	}
}
