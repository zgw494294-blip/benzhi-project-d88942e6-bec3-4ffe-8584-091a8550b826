package domain

import (
	"strings"
	"time"
)

type TermPack struct {
	ID              string         `json:"id"`
	ConferenceName  string         `json:"conferenceName"`
	SourceLanguage  string         `json:"sourceLanguage"`
	TargetLanguage  string         `json:"targetLanguage"`
	Status          TermPackStatus `json:"status"`
	CurrentRevision int            `json:"currentRevision"`
	FrozenRevision  int            `json:"frozenRevision"`
	Version         uint64         `json:"version"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

func NewTermPack(id, conference, sourceLanguage, targetLanguage string, now time.Time) (TermPack, error) {
	p := TermPack{ID: strings.TrimSpace(id), ConferenceName: strings.TrimSpace(conference), SourceLanguage: strings.TrimSpace(sourceLanguage), TargetLanguage: strings.TrimSpace(targetLanguage), Status: StatusDraft, CurrentRevision: 1, Version: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
	if p.ID == "" || p.ConferenceName == "" || p.SourceLanguage == "" || p.TargetLanguage == "" {
		return TermPack{}, NewRuleError("invalid_pack", "会议主题、源语言和目标语言不能为空")
	}
	if strings.EqualFold(p.SourceLanguage, p.TargetLanguage) {
		return TermPack{}, NewRuleError("invalid_language_pair", "源语言和目标语言不能相同")
	}
	return p, nil
}

func (p *TermPack) RequireVersion(expected uint64) error {
	if expected == 0 || p.Version != expected {
		return ErrVersionConflict
	}
	return nil
}

func (p *TermPack) Mutable() error {
	if p.Status == StatusReleased {
		return NewRuleError("released_immutable", "已发布术语包不可修改")
	}
	return nil
}

func (p *TermPack) Advance(to TermPackStatus, now time.Time) error {
	if err := p.Mutable(); err != nil {
		return err
	}
	if err := RequireTransition(p.Status, to); err != nil {
		return err
	}
	p.Status = to
	p.Version++
	p.UpdatedAt = now.UTC()
	return nil
}

func (p *TermPack) Touch(now time.Time) error {
	if err := p.Mutable(); err != nil {
		return err
	}
	p.Version++
	p.UpdatedAt = now.UTC()
	return nil
}

func (p *TermPack) UpdateMetadata(conference, sourceLanguage, targetLanguage string, expected uint64, now time.Time) error {
	if err := p.RequireVersion(expected); err != nil {
		return err
	}
	if p.Status != StatusDraft {
		return NewRuleError("invalid_state", "只有草稿可以修改术语包元数据")
	}
	conference = strings.TrimSpace(conference)
	sourceLanguage = strings.TrimSpace(sourceLanguage)
	targetLanguage = strings.TrimSpace(targetLanguage)
	if conference == "" || sourceLanguage == "" || targetLanguage == "" {
		return NewRuleError("invalid_pack", "会议主题、源语言和目标语言不能为空")
	}
	if strings.EqualFold(sourceLanguage, targetLanguage) {
		return NewRuleError("invalid_language_pair", "源语言和目标语言不能相同")
	}
	p.ConferenceName = conference
	p.SourceLanguage = sourceLanguage
	p.TargetLanguage = targetLanguage
	return p.Touch(now)
}

func (p *TermPack) Submit(entries []TermEntry, now time.Time) error {
	if p.Status != StatusDraft {
		return NewRuleError("invalid_state", "只有草稿可以提交审定")
	}
	if len(entries) == 0 {
		return NewRuleError("empty_pack", "至少编制一个完整词条后才能提交")
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.Revision != p.CurrentRevision || entry.TermPackID != p.ID {
			return NewRuleError("revision_mismatch", "词条不属于当前修订")
		}
		if err := entry.ValidateDraft(); err != nil {
			return err
		}
		key := strings.ToLower(entry.SourceTerm)
		if seen[key] {
			return NewRuleError("duplicate_term", "当前修订存在重复源术语")
		}
		seen[key] = true
	}
	return p.Advance(StatusSubmitted, now)
}

func (p *TermPack) CompleteReview(entries []TermEntry, now time.Time) error {
	if p.Status != StatusSubmitted {
		return NewRuleError("invalid_state", "术语包当前不在审定阶段")
	}
	if len(entries) == 0 {
		return NewRuleError("empty_pack", "没有可审定词条")
	}
	for _, entry := range entries {
		if !entry.Approved() {
			return NewRuleError("review_incomplete", "全部词条接受或替换后才能完成审定")
		}
	}
	return p.Advance(StatusReviewed, now)
}

func (p *TermPack) Freeze(now time.Time) error {
	if p.Status != StatusReviewed {
		return NewRuleError("invalid_state", "只有已完成审定的修订可以冻结")
	}
	p.FrozenRevision = p.CurrentRevision
	return p.Advance(StatusFrozen, now)
}

func (p *TermPack) StartRehearsal(now time.Time) error {
	return p.Advance(StatusRehearsal, now)
}

func (p *TermPack) RequestChanges(now time.Time) error {
	if p.Status == StatusChangesRequested {
		return p.Touch(now)
	}
	return p.Advance(StatusChangesRequested, now)
}

func (p *TermPack) BeginRevision(now time.Time) error {
	if p.Status != StatusChangesRequested {
		return NewRuleError("invalid_state", "只有要求修改的术语包可以创建新修订")
	}
	p.CurrentRevision++
	p.FrozenRevision = 0
	return p.Advance(StatusDraft, now)
}

func (p *TermPack) Release(now time.Time) error {
	if p.Status != StatusRehearsal {
		return NewRuleError("invalid_state", "只有完成演练的候选版可以发布")
	}
	return p.Advance(StatusReleased, now)
}
