package missingfindingentryreport

import (
	"context"
	"database/sql"
	"testing"

	"termpack/internal/application"
	"termpack/internal/domain"
	"termpack/internal/store/sqlite"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestMissingFindingEntryDoesNotCrashReport(t *testing.T) {
	dbPath := t.TempDir() + "/term-packs.db"
	store, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.NewService(store)
	ctx := context.Background()

	view, err := service.Create(ctx, application.CreatePack{ConferenceName: "口译演练", SourceLanguage: "中文", TargetLanguage: "英语", IdempotencyKey: "create"})
	if err != nil {
		t.Fatal(err)
	}
	packID := view.Pack.ID
	view, err = service.AddEntry(ctx, application.AddEntry{PackID: packID, SourceTerm: "术语", PreferredTranslation: "term", Definition: "定义", Context: "演练", Evidence: "议程", ExpectedVersion: view.Pack.Version, IdempotencyKey: "entry"})
	if err != nil {
		t.Fatal(err)
	}
	entryID := view.Entries[0].ID
	view, err = service.Submit(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "submit"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.ReviewEntry(ctx, application.ReviewEntry{PackID: packID, EntryID: entryID, Decision: domain.DecisionAccepted, ExpectedVersion: view.Pack.Version, IdempotencyKey: "review"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.Freeze(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "freeze"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.StartRehearsal(ctx, application.VersionedCommand{PackID: packID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "rehearsal"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.AddFinding(ctx, application.AddFinding{PackID: packID, EntryID: entryID, Scenario: "问答", Severity: domain.SeverityMajor, Observation: "译法不清", ExpectedVersion: view.Pack.Version, IdempotencyKey: "finding"})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Findings) != 1 {
		t.Fatal("应先登记一条演练发现")
	}

	otherDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := otherDB.ExecContext(ctx, "DELETE FROM term_entries WHERE id = ?", entryID); err != nil {
		otherDB.Close()
		t.Fatal(err)
	}
	if err := otherDB.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = service.FindingsReport(ctx, application.FindingFilter{PackID: packID})
	if err == nil {
		t.Fatal("词条资源失效时报告应返回可识别的数据完整性错误")
	}
}
