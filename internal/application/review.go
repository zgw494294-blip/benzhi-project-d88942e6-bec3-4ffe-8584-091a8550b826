package application

import (
	"context"

	"termpack/internal/domain"
)

func (s *Service) ReviewEntry(ctx context.Context, cmd ReviewEntry) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "entry.reviewed", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		if pack.Status != domain.StatusSubmitted {
			return PackView{}, domain.NewRuleError("invalid_state", "只有已提交词条可以审定")
		}
		entry, err := repo.GetEntry(ctx, pack.ID, cmd.EntryID, pack.CurrentRevision)
		if err != nil {
			return PackView{}, err
		}
		if entry.Decision != domain.DecisionPending {
			return PackView{}, domain.NewRuleError("entry_reviewed", "词条已经处理，不能重复审定")
		}
		if err := entry.Review(cmd.Decision, cmd.Translation, cmd.EditorNote); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdateEntry(ctx, entry, false); err != nil {
			return PackView{}, err
		}
		entries, err := repo.EntriesForRevision(ctx, pack.ID, pack.CurrentRevision)
		if err != nil {
			return PackView{}, err
		}
		from, old := pack.Status, pack.Version
		complete := true
		for _, candidate := range entries {
			if !candidate.Approved() {
				complete = false
				break
			}
		}
		if complete {
			err = pack.CompleteReview(entries, s.now())
		} else {
			err = pack.Touch(s.now())
		}
		if err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePack(ctx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, "entry.reviewed", from, cmd); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) ReviewEntries(ctx context.Context, cmd BatchReview) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "entries.batch_reviewed", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		if pack.Status != domain.StatusSubmitted {
			return PackView{}, domain.NewRuleError("invalid_state", "只有已提交词条可以批量审定")
		}
		if len(cmd.Items) == 0 {
			return PackView{}, domain.NewRuleError("empty_review", "批量审定至少提交一条词条")
		}
		seen := make(map[string]bool, len(cmd.Items))
		entries := make([]domain.TermEntry, 0, len(cmd.Items))
		for _, item := range cmd.Items {
			if seen[item.EntryID] {
				return PackView{}, domain.NewDetailedRuleError("duplicate_entry", "批量审定中不能重复提交同一词条", item.EntryID)
			}
			seen[item.EntryID] = true
			entry, getErr := repo.GetEntry(ctx, pack.ID, item.EntryID, pack.CurrentRevision)
			if getErr != nil {
				return PackView{}, getErr
			}
			if entry.Decision != domain.DecisionPending {
				return PackView{}, domain.NewRuleError("entry_reviewed", "只能处理仍为 Pending 的词条")
			}
			if item.Decision != domain.DecisionAccepted && item.Decision != domain.DecisionReplaced {
				return PackView{}, domain.NewRuleError("invalid_decision", "批量审定只允许 Accepted 或 Replaced")
			}
			if err := entry.Review(item.Decision, item.Translation, item.EditorNote); err != nil {
				return PackView{}, err
			}
			entries = append(entries, entry)
		}
		for _, entry := range entries {
			if err := repo.UpdateEntry(ctx, entry, false); err != nil {
				return PackView{}, err
			}
		}
		allEntries, err := repo.EntriesForRevision(ctx, pack.ID, pack.CurrentRevision)
		if err != nil {
			return PackView{}, err
		}
		from, old := pack.Status, pack.Version
		complete := len(allEntries) > 0
		for _, entry := range allEntries {
			if !entry.Approved() {
				complete = false
				break
			}
		}
		if complete {
			if err := pack.CompleteReview(allEntries, s.now()); err != nil {
				return PackView{}, err
			}
		} else if err := pack.Touch(s.now()); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePack(ctx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, "entries.batch_reviewed", from, map[string]any{"items": cmd.Items, "completed": complete}); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) Freeze(ctx context.Context, cmd VersionedCommand) (PackView, error) {
	return s.transition(ctx, cmd, "pack.frozen", func(repo Repository, pack *domain.TermPack) error {
		entries, err := repo.EntriesForRevision(ctx, pack.ID, pack.CurrentRevision)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.Approved() {
				return domain.NewRuleError("review_incomplete", "存在未通过审定的词条")
			}
		}
		return pack.Freeze(s.now())
	})
}

func (s *Service) StartRehearsal(ctx context.Context, cmd VersionedCommand) (PackView, error) {
	return s.transition(ctx, cmd, "rehearsal.started", func(_ Repository, pack *domain.TermPack) error {
		return pack.StartRehearsal(s.now())
	})
}
