package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"termpack/internal/domain"
)

func (s *Service) Preflight(ctx context.Context, packID string, revision int) (PreflightReport, error) {
	var report PreflightReport
	err := s.store.View(ctx, func(repo Repository) error {
		pack, err := repo.GetPack(ctx, packID)
		if err != nil {
			return err
		}
		if revision == 0 {
			revision = pack.CurrentRevision
		}
		if revision < 1 || revision > pack.CurrentRevision {
			return domain.ErrNotFound
		}
		entries, err := repo.EntriesForRevision(ctx, pack.ID, revision)
		if err != nil {
			return err
		}
		report = PreflightReport{TermPackID: pack.ID, Revision: revision, Problems: []PreflightProblem{}}
		seen := map[string]string{}
		for _, entry := range entries {
			report.Total++
			switch entry.Decision {
			case domain.DecisionPending:
				report.Pending++
			case domain.DecisionAccepted:
				report.Accepted++
			case domain.DecisionReplaced:
				report.Replaced++
			case domain.DecisionRejected:
				report.Rejected++
			}
			if entry.TermPackID != pack.ID || entry.Revision != revision {
				report.Problems = append(report.Problems, PreflightProblem{EntryID: entry.ID, Field: "revision", Code: "revision_mismatch", Message: "词条不属于指定修订"})
			}
			if err := entry.ValidateDraft(); err != nil {
				report.Problems = append(report.Problems, PreflightProblem{EntryID: entry.ID, Field: "entry", Code: domain.ErrorCode(err), Message: err.Error()})
			}
			key := strings.ToLower(strings.TrimSpace(entry.SourceTerm))
			if previous, ok := seen[key]; ok {
				report.Problems = append(report.Problems, PreflightProblem{EntryID: entry.ID, Field: "sourceTerm", Code: "duplicate_term", Message: fmt.Sprintf("与词条 %s 的源术语重复", previous)})
			} else {
				seen[key] = entry.ID
			}
		}
		report.CanSubmit = pack.Status == domain.StatusDraft && report.Total > 0 && len(report.Problems) == 0
		report.CanCompleteReview = pack.Status == domain.StatusSubmitted && report.Total > 0 && report.Pending == 0 && report.Rejected == 0 && len(report.Problems) == 0
		return nil
	})
	return report, err
}

func (s *Service) RevisionDiff(ctx context.Context, packID string, currentRevision, previousRevision int) (RevisionDiff, error) {
	var diff RevisionDiff
	err := s.store.View(ctx, func(repo Repository) error {
		pack, err := repo.GetPack(ctx, packID)
		if err != nil {
			return err
		}
		if currentRevision == 0 {
			currentRevision = pack.CurrentRevision
		}
		if previousRevision == 0 {
			previousRevision = currentRevision - 1
		}
		if currentRevision < 1 || previousRevision < 1 || currentRevision > pack.CurrentRevision || previousRevision >= currentRevision {
			return domain.ErrNotFound
		}
		current, err := repo.EntriesForRevision(ctx, pack.ID, currentRevision)
		if err != nil {
			return err
		}
		previous, err := repo.EntriesForRevision(ctx, pack.ID, previousRevision)
		if err != nil {
			return err
		}
		if len(current) == 0 && len(previous) == 0 {
			return domain.ErrNotFound
		}
		previousByTerm := make(map[string]domain.TermEntry, len(previous))
		for _, entry := range previous {
			previousByTerm[strings.ToLower(strings.TrimSpace(entry.SourceTerm))] = entry
		}
		currentByTerm := make(map[string]domain.TermEntry, len(current))
		for _, entry := range current {
			currentByTerm[strings.ToLower(strings.TrimSpace(entry.SourceTerm))] = entry
		}
		keys := make(map[string]bool, len(previousByTerm)+len(currentByTerm))
		for key := range previousByTerm {
			keys[key] = true
		}
		for key := range currentByTerm {
			keys[key] = true
		}
		ordered := make([]string, 0, len(keys))
		for key := range keys {
			ordered = append(ordered, key)
		}
		sort.Strings(ordered)
		diff = RevisionDiff{TermPackID: pack.ID, PreviousRevision: previousRevision, CurrentRevision: currentRevision, Items: make([]DiffItem, 0, len(ordered))}
		for _, key := range ordered {
			oldEntry, hadOld := previousByTerm[key]
			newEntry, hadNew := currentByTerm[key]
			item := DiffItem{}
			switch {
			case !hadOld:
				item.Classification, item.SourceTerm = "Added", newEntry.SourceTerm
				item.Current = &newEntry
			case !hadNew:
				item.Classification, item.SourceTerm = "Removed", oldEntry.SourceTerm
				item.Previous = &oldEntry
			default:
				item.SourceTerm = newEntry.SourceTerm
				item.Previous, item.Current = &oldEntry, &newEntry
				if sameEntryContent(oldEntry, newEntry) {
					item.Classification = "Unchanged"
				} else {
					item.Classification = "Changed"
				}
			}
			diff.Items = append(diff.Items, item)
		}
		return nil
	})
	return diff, err
}

func sameEntryContent(a, b domain.TermEntry) bool {
	return a.SourceTerm == b.SourceTerm && a.PreferredTranslation == b.PreferredTranslation && a.Definition == b.Definition && a.Context == b.Context && a.Evidence == b.Evidence && a.Decision == b.Decision && a.EditorNote == b.EditorNote
}

type certificateSnapshot struct {
	TermPackID       string             `json:"termPackID"`
	ConferenceName   string             `json:"conferenceName"`
	SourceLanguage   string             `json:"sourceLanguage"`
	TargetLanguage   string             `json:"targetLanguage"`
	ReleasedRevision int                `json:"releasedRevision"`
	Entries          []domain.TermEntry `json:"entries"`
}

func (s *Service) VerifyCertificate(ctx context.Context, packID string) (CertificateVerification, error) {
	if cached, ok := s.cachedCertificateVerification(packID); ok {
		return cached, nil
	}
	var result CertificateVerification
	err := s.store.View(ctx, func(repo Repository) error {
		pack, err := repo.GetPack(ctx, packID)
		if err != nil {
			return err
		}
		if pack.Status != domain.StatusReleased {
			return domain.NewRuleError("invalid_state", "只有已发布术语包可以核验发布凭据")
		}
		certificate, err := repo.Certificate(ctx, pack.ID)
		if err != nil {
			return err
		}
		if certificate == nil {
			return domain.ErrNotFound
		}
		entries, err := repo.EntriesForRevision(ctx, pack.ID, certificate.ReleasedRevision)
		if err != nil {
			return err
		}
		result = CertificateVerification{TermPackID: pack.ID, Checks: []CertificateCheck{}}
		add := func(name string, valid bool, message string) {
			result.Checks = append(result.Checks, CertificateCheck{Name: name, Valid: valid, Message: message})
			if !valid {
				result.Valid = false
			}
		}
		digest := sha256.Sum256(certificate.SnapshotJSON)
		computed := "sha256:" + hex.EncodeToString(digest[:])
		add("contentDigest", computed == certificate.ContentDigest, "contentDigest 与 SnapshotJSON 一致")
		var snapshot certificateSnapshot
		if unmarshalErr := json.Unmarshal(certificate.SnapshotJSON, &snapshot); unmarshalErr != nil {
			add("snapshotJSON", false, "SnapshotJSON 不是有效快照")
			return nil
		}
		add("termPackID", snapshot.TermPackID == pack.ID && certificate.TermPackID == pack.ID, "术语包标识一致")
		add("releasedRevision", snapshot.ReleasedRevision == certificate.ReleasedRevision && certificate.ReleasedRevision == pack.CurrentRevision, "发布修订一致")
		add("entryCount", snapshot.EntryCount() == certificate.EntryCount && certificate.EntryCount == len(entries), "词条数量一致")
		add("languageMetadata", snapshot.ConferenceName == pack.ConferenceName && snapshot.SourceLanguage == pack.SourceLanguage && snapshot.TargetLanguage == pack.TargetLanguage, "会议与语言元数据一致")
		approved := true
		byID := make(map[string]domain.TermEntry, len(entries))
		for _, entry := range entries {
			byID[entry.ID] = entry
			if !entry.Approved() {
				approved = false
			}
		}
		if len(snapshot.Entries) != len(entries) {
			approved = false
		}
		for _, entry := range snapshot.Entries {
			dbEntry, ok := byID[entry.ID]
			if !ok || !entry.Approved() || !dbEntry.Approved() || !sameEntryContent(entry, dbEntry) {
				approved = false
				break
			}
		}
		add("approvedEntries", approved, "快照词条均已审定且与数据库一致")
		result.Valid = true
		for _, check := range result.Checks {
			if !check.Valid {
				result.Valid = false
				break
			}
		}
		return nil
	})
	if err == nil {
		s.cacheCertificateVerification(packID, result)
	}
	return result, err
}

func (s certificateSnapshot) EntryCount() int { return len(s.Entries) }
