package domain

import "fmt"

type TermPackStatus string

const (
	StatusDraft            TermPackStatus = "Draft"
	StatusSubmitted        TermPackStatus = "Submitted"
	StatusReviewed         TermPackStatus = "Reviewed"
	StatusFrozen           TermPackStatus = "Frozen"
	StatusRehearsal        TermPackStatus = "Rehearsal"
	StatusChangesRequested TermPackStatus = "ChangesRequested"
	StatusReleased         TermPackStatus = "Released"
)

var transitions = map[TermPackStatus]map[TermPackStatus]bool{
	StatusDraft:            {StatusSubmitted: true},
	StatusSubmitted:        {StatusReviewed: true},
	StatusReviewed:         {StatusFrozen: true},
	StatusFrozen:           {StatusRehearsal: true},
	StatusRehearsal:        {StatusChangesRequested: true, StatusReleased: true},
	StatusChangesRequested: {StatusDraft: true},
}

func (s TermPackStatus) Valid() bool {
	_, ok := transitions[s]
	return ok || s == StatusReleased
}

func CanTransition(from, to TermPackStatus) bool {
	return transitions[from][to]
}

func RequireTransition(from, to TermPackStatus) error {
	if !CanTransition(from, to) {
		return NewRuleError("invalid_transition", fmt.Sprintf("术语包不能从 %s 进入 %s", from, to))
	}
	return nil
}
