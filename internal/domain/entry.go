package domain

import "strings"

type EntryDecision string

const (
	DecisionPending  EntryDecision = "Pending"
	DecisionAccepted EntryDecision = "Accepted"
	DecisionReplaced EntryDecision = "Replaced"
	DecisionRejected EntryDecision = "Rejected"
)

type TermEntry struct {
	ID                   string        `json:"id"`
	TermPackID           string        `json:"termPackID"`
	Revision             int           `json:"revision"`
	SourceTerm           string        `json:"sourceTerm"`
	PreferredTranslation string        `json:"preferredTranslation"`
	Definition           string        `json:"definition"`
	Context              string        `json:"context"`
	Evidence             string        `json:"evidence"`
	Decision             EntryDecision `json:"decision"`
	EditorNote           string        `json:"editorNote"`
}

func NewEntry(id, packID string, revision int, source, translation, definition, context, evidence string) (TermEntry, error) {
	e := TermEntry{ID: strings.TrimSpace(id), TermPackID: packID, Revision: revision, SourceTerm: strings.TrimSpace(source), PreferredTranslation: strings.TrimSpace(translation), Definition: strings.TrimSpace(definition), Context: strings.TrimSpace(context), Evidence: strings.TrimSpace(evidence), Decision: DecisionPending}
	if err := e.ValidateDraft(); err != nil {
		return TermEntry{}, err
	}
	return e, nil
}

func (e TermEntry) ValidateDraft() error {
	switch {
	case e.ID == "" || e.TermPackID == "":
		return NewRuleError("invalid_entry", "词条标识不能为空")
	case e.Revision < 1:
		return NewRuleError("invalid_entry", "词条修订号必须大于零")
	case strings.TrimSpace(e.SourceTerm) == "":
		return NewRuleError("invalid_entry", "源术语不能为空")
	case strings.TrimSpace(e.PreferredTranslation) == "":
		return NewRuleError("invalid_entry", "首选译法不能为空")
	case strings.TrimSpace(e.Definition) == "":
		return NewRuleError("invalid_entry", "定义不能为空")
	case strings.TrimSpace(e.Context) == "":
		return NewRuleError("invalid_entry", "语境不能为空")
	case strings.TrimSpace(e.Evidence) == "":
		return NewRuleError("invalid_entry", "来源依据不能为空")
	}
	return nil
}

func (e *TermEntry) Review(decision EntryDecision, translation, note string) error {
	if decision != DecisionAccepted && decision != DecisionReplaced && decision != DecisionRejected {
		return NewRuleError("invalid_decision", "审定结论必须为 Accepted、Replaced 或 Rejected")
	}
	translation = strings.TrimSpace(translation)
	note = strings.TrimSpace(note)
	if decision == DecisionReplaced && translation == "" {
		return NewRuleError("invalid_decision", "替换词条必须提供新的首选译法")
	}
	if (decision == DecisionReplaced || decision == DecisionRejected) && note == "" {
		return NewRuleError("invalid_decision", "替换或驳回词条必须填写编辑说明")
	}
	if decision == DecisionReplaced {
		e.PreferredTranslation = translation
	}
	e.Decision = decision
	e.EditorNote = note
	return nil
}

func (e *TermEntry) Edit(source, translation, definition, context, evidence string) error {
	if e.Decision != DecisionPending {
		return NewRuleError("entry_reviewed", "已审定词条不能直接编辑")
	}
	candidate, err := NewEntry(e.ID, e.TermPackID, e.Revision, source, translation, definition, context, evidence)
	if err != nil {
		return err
	}
	*e = candidate
	return nil
}

func (e TermEntry) Approved() bool {
	return e.Decision == DecisionAccepted || e.Decision == DecisionReplaced
}

func (e TermEntry) ForRevision(id string, revision int) TermEntry {
	e.ID = id
	e.Revision = revision
	e.Decision = DecisionPending
	e.EditorNote = ""
	return e
}
