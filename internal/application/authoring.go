package application

import (
	"context"
	"strings"

	"termpack/internal/domain"
)

func (s *Service) UpdateMetadata(ctx context.Context, cmd UpdateMetadata) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "pack.metadata.updated", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		old := pack
		if err := pack.UpdateMetadata(cmd.ConferenceName, cmd.SourceLanguage, cmd.TargetLanguage, cmd.ExpectedVersion, s.now()); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePackMetadata(ctx, pack, old.Version); err != nil {
			return PackView{}, err
		}
		payload := map[string]any{
			"old": map[string]string{"conferenceName": old.ConferenceName, "sourceLanguage": old.SourceLanguage, "targetLanguage": old.TargetLanguage},
			"new": map[string]string{"conferenceName": pack.ConferenceName, "sourceLanguage": pack.SourceLanguage, "targetLanguage": pack.TargetLanguage},
		}
		if err := s.audit(ctx, repo, pack, "pack.metadata.updated", old.Status, payload); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) Create(ctx context.Context, cmd CreatePack) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "pack.created", "", func(repo Repository) (PackView, error) {
		pack, err := domain.NewTermPack(newID(), cmd.ConferenceName, cmd.SourceLanguage, cmd.TargetLanguage, s.now())
		if err != nil {
			return PackView{}, err
		}
		if err := repo.InsertPack(ctx, pack); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, "pack.created", "", cmd); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) AddEntry(ctx context.Context, cmd AddEntry) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "entry.added", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		if pack.Status != domain.StatusDraft {
			return PackView{}, domain.NewRuleError("invalid_state", "只有草稿修订可以编制词条")
		}
		entry, err := domain.NewEntry(newID(), pack.ID, pack.CurrentRevision, cmd.SourceTerm, cmd.PreferredTranslation, cmd.Definition, cmd.Context, cmd.Evidence)
		if err != nil {
			return PackView{}, err
		}
		if err := repo.InsertEntry(ctx, entry); err != nil {
			return PackView{}, err
		}
		old := pack.Version
		if err := pack.Touch(s.now()); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePack(ctx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, "entry.added", domain.StatusDraft, entry); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) AddEntries(ctx context.Context, cmd BatchAddEntries) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "entries.batch_added", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		if pack.Status != domain.StatusDraft {
			return PackView{}, domain.NewRuleError("invalid_state", "只有草稿修订可以编制词条")
		}
		if len(cmd.Entries) == 0 {
			return PackView{}, domain.NewRuleError("empty_entries", "批量编制至少提交一条词条")
		}
		existing, err := repo.EntriesForRevision(ctx, pack.ID, pack.CurrentRevision)
		if err != nil {
			return PackView{}, err
		}
		type seenEntry struct {
			id    string
			index int
		}
		seen := make(map[string]seenEntry, len(existing)+len(cmd.Entries))
		for _, entry := range existing {
			seen[strings.ToLower(strings.TrimSpace(entry.SourceTerm))] = seenEntry{id: entry.ID, index: -1}
		}
		conflicts := []map[string]any{}
		entries := make([]domain.TermEntry, 0, len(cmd.Entries))
		for index, input := range cmd.Entries {
			entry, entryErr := domain.NewEntry(newID(), pack.ID, pack.CurrentRevision, input.SourceTerm, input.PreferredTranslation, input.Definition, input.Context, input.Evidence)
			if entryErr != nil {
				conflicts = append(conflicts, map[string]any{"index": index, "code": domain.ErrorCode(entryErr), "message": entryErr.Error()})
				continue
			}
			normalized := strings.ToLower(strings.TrimSpace(entry.SourceTerm))
			if previous, ok := seen[normalized]; ok {
				detail := map[string]any{"index": index, "sourceTerm": entry.SourceTerm, "conflictsWithEntryID": previous.id, "code": "duplicate_term", "message": "当前修订存在重复源术语"}
				if previous.index >= 0 {
					detail["conflictsWithIndex"] = previous.index
				}
				conflicts = append(conflicts, detail)
				continue
			}
			seen[normalized] = seenEntry{id: entry.ID, index: index}
			entries = append(entries, entry)
		}
		if len(conflicts) > 0 {
			return PackView{}, domain.NewDetailedRuleError("batch_validation_failed", "批量词条校验失败，未写入任何词条", conflicts)
		}
		for _, entry := range entries {
			if err := repo.InsertEntry(ctx, entry); err != nil {
				return PackView{}, err
			}
		}
		old := pack.Version
		if err := pack.Touch(s.now()); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePack(ctx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, "entries.batch_added", domain.StatusDraft, map[string]any{"entries": entries}); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) UpdateEntry(ctx context.Context, cmd UpdateEntry) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "entry.updated", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		if pack.Status != domain.StatusDraft {
			return PackView{}, domain.NewRuleError("invalid_state", "只有草稿修订可以编辑词条")
		}
		entry, err := repo.GetEntry(ctx, pack.ID, cmd.EntryID, pack.CurrentRevision)
		if err != nil {
			return PackView{}, err
		}
		if err := entry.Edit(cmd.SourceTerm, cmd.PreferredTranslation, cmd.Definition, cmd.Context, cmd.Evidence); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdateEntry(ctx, entry, true); err != nil {
			return PackView{}, err
		}
		old := pack.Version
		if err := pack.Touch(s.now()); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePack(ctx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, "entry.updated", domain.StatusDraft, entry); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) Submit(ctx context.Context, cmd VersionedCommand) (PackView, error) {
	return s.transition(ctx, cmd, "pack.submitted", func(repo Repository, pack *domain.TermPack) error {
		entries, err := repo.EntriesForRevision(ctx, pack.ID, pack.CurrentRevision)
		if err != nil {
			return err
		}
		return pack.Submit(entries, s.now())
	})
}

func (s *Service) transition(ctx context.Context, cmd VersionedCommand, action string, change func(Repository, *domain.TermPack) error) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, action, cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		from, old := pack.Status, pack.Version
		if err := change(repo, &pack); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePack(ctx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, action, from, cmd); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}
