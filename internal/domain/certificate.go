package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

type ReleaseCertificate struct {
	ID               string          `json:"id"`
	TermPackID       string          `json:"termPackID"`
	ReleasedRevision int             `json:"releasedRevision"`
	EntryCount       int             `json:"entryCount"`
	ApprovedBy       string          `json:"approvedBy"`
	ApprovedAt       time.Time       `json:"approvedAt"`
	ContentDigest    string          `json:"contentDigest"`
	SnapshotJSON     json.RawMessage `json:"snapshotJSON"`
}

type releaseSnapshot struct {
	TermPackID       string      `json:"termPackID"`
	ConferenceName   string      `json:"conferenceName"`
	SourceLanguage   string      `json:"sourceLanguage"`
	TargetLanguage   string      `json:"targetLanguage"`
	ReleasedRevision int         `json:"releasedRevision"`
	Entries          []TermEntry `json:"entries"`
}

func BuildCertificate(id, approvedBy string, pack TermPack, entries []TermEntry, now time.Time) (ReleaseCertificate, error) {
	approvedBy = strings.TrimSpace(approvedBy)
	if id == "" || approvedBy == "" {
		return ReleaseCertificate{}, NewRuleError("invalid_approval", "发布负责人不能为空")
	}
	approved := append([]TermEntry(nil), entries...)
	sort.Slice(approved, func(i, j int) bool {
		if approved[i].SourceTerm == approved[j].SourceTerm {
			return approved[i].ID < approved[j].ID
		}
		return approved[i].SourceTerm < approved[j].SourceTerm
	})
	for _, entry := range approved {
		if entry.Revision != pack.CurrentRevision || !entry.Approved() {
			return ReleaseCertificate{}, NewRuleError("release_blocked", "发布快照包含未审定或修订不一致的词条")
		}
	}
	snapshot, err := json.Marshal(releaseSnapshot{TermPackID: pack.ID, ConferenceName: pack.ConferenceName, SourceLanguage: pack.SourceLanguage, TargetLanguage: pack.TargetLanguage, ReleasedRevision: pack.CurrentRevision, Entries: approved})
	if err != nil {
		return ReleaseCertificate{}, err
	}
	digest := sha256.Sum256(snapshot)
	return ReleaseCertificate{ID: id, TermPackID: pack.ID, ReleasedRevision: pack.CurrentRevision, EntryCount: len(approved), ApprovedBy: approvedBy, ApprovedAt: now.UTC(), ContentDigest: "sha256:" + hex.EncodeToString(digest[:]), SnapshotJSON: snapshot}, nil
}
