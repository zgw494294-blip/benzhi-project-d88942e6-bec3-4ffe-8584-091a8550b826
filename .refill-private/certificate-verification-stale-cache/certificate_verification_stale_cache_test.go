package certificateverificationstalecache_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"termpack/internal/application"
	"termpack/internal/domain"
	"termpack/internal/store/sqlite"
)

func TestCertificateVerificationRefreshesPersistedEvidence(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "termpack.db")
	store, err := sqlite.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := application.NewService(store)
	ctx := context.Background()
	view, err := service.Create(ctx, application.CreatePack{
		ConferenceName: "缓存一致性会议",
		SourceLanguage: "中文",
		TargetLanguage: "英语",
		IdempotencyKey: "stale-cache-create",
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.AddEntry(ctx, application.AddEntry{
		PackID:               view.Pack.ID,
		SourceTerm:           "发布凭据",
		PreferredTranslation: "release certificate",
		Definition:           "不可变发布证明",
		Context:              "发布核验",
		Evidence:             "术语工作台规范",
		ExpectedVersion:      view.Pack.Version,
		IdempotencyKey:       "stale-cache-add",
	})
	if err != nil {
		t.Fatal(err)
	}
	entryID := view.Entries[0].ID
	view, err = service.Submit(ctx, application.VersionedCommand{PackID: view.Pack.ID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "stale-cache-submit"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.ReviewEntry(ctx, application.ReviewEntry{PackID: view.Pack.ID, EntryID: entryID, Decision: domain.DecisionAccepted, ExpectedVersion: view.Pack.Version, IdempotencyKey: "stale-cache-review"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.Freeze(ctx, application.VersionedCommand{PackID: view.Pack.ID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "stale-cache-freeze"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.StartRehearsal(ctx, application.VersionedCommand{PackID: view.Pack.ID, ExpectedVersion: view.Pack.Version, IdempotencyKey: "stale-cache-rehearsal"})
	if err != nil {
		t.Fatal(err)
	}
	view, err = service.Release(ctx, application.Release{PackID: view.Pack.ID, ApprovedBy: "发布负责人", ExpectedVersion: view.Pack.Version, IdempotencyKey: "stale-cache-release"})
	if err != nil {
		t.Fatal(err)
	}

	first, err := service.VerifyCertificate(ctx, view.Pack.ID)
	if err != nil || !first.Valid {
		t.Fatalf("初次核验应确认发布凭据有效: valid=%v err=%v", first.Valid, err)
	}

	external, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer external.Close()
	if _, err := external.ExecContext(ctx, `UPDATE release_certificates SET content_digest='sha256:tampered' WHERE term_pack_id=?`, view.Pack.ID); err != nil {
		t.Fatal(err)
	}

	second, err := service.VerifyCertificate(ctx, view.Pack.ID)
	if err != nil {
		t.Fatal(err)
	}
	if second.Valid {
		t.Fatal("持久化发布凭据已被修改，重复核验不得复用旧的有效结果")
	}
}
