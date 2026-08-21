package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"termpack/internal/domain"
)

type Service struct {
	store       Store
	now         func() time.Time
	commandGate chan struct{}
}

func NewService(store Store) *Service {
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return &Service{store: store, now: time.Now, commandGate: gate}
}

func (s *Service) execute(ctx context.Context, key, action, packID string, run func(Repository) (PackView, error)) (PackView, error) {
	select {
	case <-s.commandGate:
		defer func() { s.commandGate <- struct{}{} }()
	default:
		return PackView{}, domain.NewRuleError("concurrent_command", "服务正在处理另一个命令，请稍后重试")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return PackView{}, domain.NewRuleError("idempotency_required", "idempotencyKey 不能为空")
	}
	var result PackView
	err := s.store.InTransaction(ctx, func(repo Repository) error {
		storedAction, raw, ok, err := repo.GetCommandResult(ctx, key)
		if err != nil {
			return err
		}
		if ok {
			if storedAction != action {
				return domain.NewRuleError("idempotency_mismatch", "idempotencyKey 已用于其他业务命令")
			}
			return json.Unmarshal(raw, &result)
		}
		result, err = run(repo)
		if err != nil {
			return err
		}
		raw, err = json.Marshal(result)
		if err != nil {
			return err
		}
		return repo.PutCommandResult(ctx, key, action, raw)
	})
	return result, err
}

func (s *Service) loadView(ctx context.Context, repo Repository, pack domain.TermPack) (PackView, error) {
	entries, err := repo.EntriesForRevision(ctx, pack.ID, pack.CurrentRevision)
	if err != nil {
		return PackView{}, err
	}
	history, err := repo.AllEntries(ctx, pack.ID)
	if err != nil {
		return PackView{}, err
	}
	findings, err := repo.Findings(ctx, pack.ID)
	if err != nil {
		return PackView{}, err
	}
	certificate, err := repo.Certificate(ctx, pack.ID)
	if err != nil {
		return PackView{}, err
	}
	audit, err := repo.AuditRecords(ctx, pack.ID)
	if err != nil {
		return PackView{}, err
	}
	return PackView{Pack: pack, Entries: entries, EntryHistory: history, Findings: findings, Certificate: certificate, AuditTrail: audit}, nil
}

func (s *Service) audit(ctx context.Context, repo Repository, pack domain.TermPack, action string, from domain.TermPackStatus, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return repo.AppendAudit(ctx, AuditRecord{ID: newID(), TermPackID: pack.ID, Action: action, FromStatus: string(from), ToStatus: string(pack.Status), Revision: pack.CurrentRevision, Payload: raw, CreatedAt: s.now().UTC()})
}

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value[:])
}

func (s *Service) Get(ctx context.Context, id string) (PackView, error) {
	var view PackView
	err := s.store.View(ctx, func(repo Repository) error {
		pack, err := repo.GetPack(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.loadView(ctx, repo, pack)
		return err
	})
	return view, err
}

func (s *Service) List(ctx context.Context) ([]domain.TermPack, error) {
	var packs []domain.TermPack
	err := s.store.View(ctx, func(repo Repository) error {
		var err error
		packs, err = repo.ListPacks(ctx)
		return err
	})
	return packs, err
}
