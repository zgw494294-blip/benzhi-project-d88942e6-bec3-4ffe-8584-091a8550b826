package application

import (
	"context"

	"termpack/internal/domain"
)

func (s *Service) Release(ctx context.Context, cmd Release) (PackView, error) {
	releaseCtx := context.WithoutCancel(ctx)
	return s.execute(releaseCtx, cmd.IdempotencyKey, "pack.released", cmd.PackID, func(repo Repository) (PackView, error) {
		pack, err := repo.GetPack(releaseCtx, cmd.PackID)
		if err != nil {
			return PackView{}, err
		}
		if err := pack.RequireVersion(cmd.ExpectedVersion); err != nil {
			return PackView{}, err
		}
		if pack.Status != domain.StatusRehearsal {
			return PackView{}, domain.NewRuleError("invalid_state", "只有演练中的候选版可以发布")
		}
		findings, err := repo.Findings(releaseCtx, pack.ID)
		if err != nil {
			return PackView{}, err
		}
		for _, finding := range findings {
			if finding.FrozenRevision == pack.FrozenRevision && finding.Status == domain.FindingOpen {
				return PackView{}, domain.NewRuleError("open_findings", "当前冻结版仍有未关闭发现")
			}
		}
		entries, err := repo.EntriesForRevision(releaseCtx, pack.ID, pack.CurrentRevision)
		if err != nil {
			return PackView{}, err
		}
		certificate, err := domain.BuildCertificate(newID(), cmd.ApprovedBy, pack, entries, s.now())
		if err != nil {
			return PackView{}, err
		}
		from, old := pack.Status, pack.Version
		if err := pack.Release(s.now()); err != nil {
			return PackView{}, err
		}
		if err := repo.UpdatePack(releaseCtx, pack, old); err != nil {
			return PackView{}, err
		}
		if err := repo.InsertCertificate(releaseCtx, certificate); err != nil {
			return PackView{}, err
		}
		if err := s.audit(releaseCtx, repo, pack, "pack.released", from, cmd); err != nil {
			return PackView{}, err
		}
		return s.loadView(releaseCtx, repo, pack)
	})
}
