package store_callback_error_atomicity_test

import (
	"context"
	"errors"
	"testing"

	"termpack/internal/application"
	"termpack/internal/domain"
	"termpack/internal/store/sqlite"
)

var errInjectedCallback = errors.New("injected callback failure")

type failAfterCallbackStore struct {
	inner application.Store
}

func (s *failAfterCallbackStore) InTransaction(ctx context.Context, fn func(application.Repository) error) error {
	return s.inner.InTransaction(ctx, func(repo application.Repository) error {
		if err := fn(repo); err != nil {
			return err
		}
		return errInjectedCallback
	})
}

func (s *failAfterCallbackStore) View(ctx context.Context, fn func(application.Repository) error) error {
	return s.inner.View(ctx, fn)
}

func (s *failAfterCallbackStore) Close() error {
	return s.inner.Close()
}

func TestStoreCallbackErrorsRollbackAndPropagate(t *testing.T) {
	store, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.NewService(&failAfterCallbackStore{inner: store})
	view, err := service.Create(context.Background(), application.CreatePack{
		ConferenceName: "事务原子性验证会",
		SourceLanguage: "中文",
		TargetLanguage: "英语",
		IdempotencyKey: "failed-create",
	})
	if !errors.Is(err, errInjectedCallback) {
		t.Fatalf("事务回调错误必须传回 application 调用方，实际错误: %v", err)
	}

	var persisted bool
	if err := store.View(context.Background(), func(repo application.Repository) error {
		_, queryErr := repo.GetPack(context.Background(), view.Pack.ID)
		persisted = queryErr == nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if persisted {
		t.Fatal("事务回调失败后术语包、审计和幂等结果不得提交")
	}

	viewErr := store.View(context.Background(), func(application.Repository) error {
		return domain.ErrNotFound
	})
	if !errors.Is(viewErr, domain.ErrNotFound) {
		t.Fatalf("只读回调错误必须跨 store 边界传播，实际错误: %v", viewErr)
	}
}
