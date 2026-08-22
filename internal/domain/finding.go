package domain

import (
	"strings"
	"time"
)

type FindingSeverity string
type FindingStatus string

const (
	SeverityMinor    FindingSeverity = "Minor"
	SeverityMajor    FindingSeverity = "Major"
	SeverityCritical FindingSeverity = "Critical"
	FindingOpen      FindingStatus   = "Open"
	FindingResolved  FindingStatus   = "Resolved"
)

type RehearsalFinding struct {
	ID             string          `json:"id"`
	TermPackID     string          `json:"termPackID"`
	FrozenRevision int             `json:"frozenRevision"`
	EntryID        string          `json:"entryID"`
	Scenario       string          `json:"scenario"`
	Severity       FindingSeverity `json:"severity"`
	Observation    string          `json:"observation"`
	Resolution     string          `json:"resolution"`
	Status         FindingStatus   `json:"status"`
	ReportedAt     time.Time       `json:"reportedAt"`
}

func NewFinding(id, packID, entryID, scenario string, revision int, severity FindingSeverity, observation string, now time.Time) (RehearsalFinding, error) {
	f := RehearsalFinding{ID: strings.TrimSpace(id), TermPackID: packID, EntryID: strings.TrimSpace(entryID), FrozenRevision: revision, Scenario: strings.TrimSpace(scenario), Severity: severity, Observation: strings.TrimSpace(observation), Status: FindingOpen, ReportedAt: now.UTC()}
	if f.ID == "" || f.EntryID == "" || f.Scenario == "" || f.Observation == "" {
		return RehearsalFinding{}, NewRuleError("invalid_finding", "演练发现的词条、场景和观察内容不能为空")
	}
	if severity != SeverityMinor && severity != SeverityMajor && severity != SeverityCritical {
		return RehearsalFinding{}, NewRuleError("invalid_finding", "未知的发现严重程度")
	}
	return f, nil
}

func (f *RehearsalFinding) Resolve(resolution string) error {
	if f.Status == FindingResolved {
		return NewRuleError("finding_closed", "演练发现已经关闭")
	}
	resolution = strings.TrimSpace(resolution)
	if resolution == "" {
		return NewRuleError("invalid_resolution", "关闭发现时必须填写处理记录")
	}
	f.Resolution = resolution
	f.Status = FindingResolved
	return nil
}
