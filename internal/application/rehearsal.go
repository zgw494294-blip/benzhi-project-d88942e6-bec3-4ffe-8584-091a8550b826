package application

import (
	"context"
	"strings"

	"termpack/internal/domain"
)

func (s *Service) AddFinding(ctx context.Context, cmd AddFinding) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "finding.reported", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		if pack.Status != domain.StatusRehearsal && pack.Status != domain.StatusChangesRequested {
			return PackView{}, domain.NewRuleError("invalid_state", "只有演练中的候选版可以登记发现")
		}
		if _, err := repo.GetEntry(ctx, pack.ID, cmd.EntryID, pack.FrozenRevision); err != nil {
			return PackView{}, err
		}
		finding, err := domain.NewFinding(newID(), pack.ID, cmd.EntryID, cmd.Scenario, pack.FrozenRevision, cmd.Severity, cmd.Observation, s.now())
		if err != nil {
			return PackView{}, err
		}
		if err := repo.InsertFinding(ctx, finding); err != nil {
			return PackView{}, err
		}
		from, old := pack.Status, pack.Version
		if cmd.Severity == domain.SeverityMajor || cmd.Severity == domain.SeverityCritical {
			err = pack.RequestChanges(s.now())
		} else {
			err = pack.Touch(s.now())
		}
		if err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePack(ctx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, "finding.reported", from, finding); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) ResolveFinding(ctx context.Context, cmd ResolveFinding) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "finding.resolved", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		if pack.Status != domain.StatusChangesRequested {
			return PackView{}, domain.NewRuleError("invalid_state", "当前没有待关闭的修订发现")
		}
		finding, err := repo.GetFinding(ctx, pack.ID, cmd.FindingID)
		if err != nil {
			return PackView{}, err
		}
		if err := finding.Resolve(cmd.Resolution); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdateFinding(ctx, finding); err != nil {
			return PackView{}, err
		}
		old := pack.Version
		if err := pack.Touch(s.now()); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePack(ctx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, "finding.resolved", pack.Status, finding); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) FindingsReport(ctx context.Context, filter FindingFilter) (FindingReport, error) {
	var report FindingReport
	err := s.store.View(ctx, func(repo Repository) error {
		pack, err := repo.GetPack(ctx, filter.PackID)
		if err != nil {
			return err
		}
		findings, err := repo.Findings(ctx, pack.ID)
		if err != nil {
			return err
		}
		history, err := repo.AllEntries(ctx, pack.ID)
		if err != nil {
			return err
		}
		entryByID := make(map[string]*domain.TermEntry, len(history))
		for i := range history {
			entryByID[history[i].ID] = &history[i]
		}
		if filter.FrozenRevision < 0 {
			return domain.ErrNotFound
		}
		if filter.FrozenRevision > 0 {
			validRevision := false
			for _, entry := range history {
				if entry.Revision == filter.FrozenRevision {
					validRevision = true
					break
				}
			}
			if !validRevision {
				return domain.ErrNotFound
			}
		}
		report = FindingReport{TermPackID: pack.ID, FrozenRevision: filter.FrozenRevision}
		for _, finding := range findings {
			if filter.FrozenRevision > 0 && finding.FrozenRevision != filter.FrozenRevision {
				continue
			}
			if filter.Severity != "" && finding.Severity != filter.Severity {
				continue
			}
			if filter.Status != "" && finding.Status != filter.Status {
				continue
			}
			entry, err := findingEntry(entryByID, finding)
			if err != nil {
				return err
			}
			item := FindingResult{Finding: finding, Entry: entry}
			report.Items = append(report.Items, item)
		}
		report.Total = len(report.Items)
		for _, item := range report.Items {
			if item.Finding.Status == domain.FindingOpen {
				report.Open++
			} else if item.Finding.Status == domain.FindingResolved {
				report.Closed++
			}
		}
		return nil
	})
	return report, err
}

func findingEntry(entries map[string]*domain.TermEntry, finding domain.RehearsalFinding) (*domain.TermEntry, error) {
	entry, ok := entries[finding.EntryID]
	if !ok || entry == nil {
		return nil, domain.NewDetailedRuleError("data_integrity", "演练发现引用的词条已失效，无法生成报告", map[string]string{
			"findingID": finding.ID,
			"entryID":   finding.EntryID,
		})
	}
	return entry, nil
}

func (s *Service) CloseFindings(ctx context.Context, cmd CloseFindings) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "findings.batch_resolved", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		if pack.Status != domain.StatusChangesRequested {
			return PackView{}, domain.NewRuleError("invalid_state", "只有要求修改状态可以批量关闭发现")
		}
		if len(cmd.Items) == 0 {
			return PackView{}, domain.NewRuleError("empty_findings", "批量关闭至少提交一条发现")
		}
		seen := make(map[string]bool, len(cmd.Items))
		findings := make([]domain.RehearsalFinding, 0, len(cmd.Items))
		for _, item := range cmd.Items {
			if seen[item.FindingID] {
				return PackView{}, domain.NewDetailedRuleError("duplicate_finding", "批量关闭中不能重复提交同一发现", item.FindingID)
			}
			seen[item.FindingID] = true
			finding, getErr := repo.GetFinding(ctx, pack.ID, item.FindingID)
			if getErr != nil {
				return PackView{}, getErr
			}
			if finding.FrozenRevision != pack.FrozenRevision {
				return PackView{}, domain.NewRuleError("revision_mismatch", "发现不属于当前冻结修订")
			}
			if finding.Status != domain.FindingOpen {
				return PackView{}, domain.NewRuleError("finding_closed", "只能关闭 Open 状态的发现")
			}
			if strings.TrimSpace(item.Resolution) == "" {
				return PackView{}, domain.NewRuleError("invalid_resolution", "关闭发现时必须填写处理记录")
			}
			if err := finding.Resolve(item.Resolution); err != nil {
				return PackView{}, err
			}
			findings = append(findings, finding)
		}
		for _, finding := range findings {
			if err := repo.UpdateFinding(ctx, finding); err != nil {
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
		if err := s.audit(ctx, repo, pack, "findings.batch_resolved", domain.StatusChangesRequested, map[string]any{"items": cmd.Items}); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}

func (s *Service) BeginRevision(ctx context.Context, cmd VersionedCommand) (PackView, error) {
	return s.execute(ctx, cmd.IdempotencyKey, "revision.created", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(ctx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		findings, err := repo.Findings(ctx, pack.ID)
		if err != nil {
			return PackView{}, err
		}
		for _, finding := range findings {
			if finding.FrozenRevision == pack.FrozenRevision && finding.Status == domain.FindingOpen {
				return PackView{}, domain.NewRuleError("open_findings", "关闭当前冻结版的全部发现后才能创建修订")
			}
		}
		oldEntries, err := repo.EntriesForRevision(ctx, pack.ID, pack.CurrentRevision)
		if err != nil {
			return PackView{}, err
		}
		from, old := pack.Status, pack.Version
		if err := pack.BeginRevision(s.now()); err != nil {
			return PackView{}, err
		}
		for _, entry := range oldEntries {
			if err := repo.InsertEntry(ctx, entry.ForRevision(newID(), pack.CurrentRevision)); err != nil {
				return PackView{}, err
			}
		}
		if err := repo.UpdatePack(ctx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := s.audit(ctx, repo, pack, "revision.created", from, cmd); err != nil {
			return PackView{}, err
		}
		return s.loadView(ctx, repo, pack)
	})
}
