package domain

import (
	"errors"
	"testing"
	"time"
)

func TestTermPackWorkflowRules(t *testing.T) {
	now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	pack, err := NewTermPack("pack-1", "全球能源会议", "中文", "英语", now)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := NewEntry("entry-1", pack.ID, 1, "虚拟电厂", "virtual power plant", "聚合能源系统", "电网专题", "背景材料")
	if err != nil {
		t.Fatal(err)
	}
	if err := pack.Submit([]TermEntry{entry}, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := pack.Freeze(now); ErrorCode(err) != "invalid_state" {
		t.Fatalf("未完成审定时应拒绝冻结，实际错误 %v", err)
	}
	if err := entry.Review(DecisionReplaced, "aggregated virtual power plant", "与演讲稿统一"); err != nil {
		t.Fatal(err)
	}
	if err := pack.CompleteReview([]TermEntry{entry}, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := pack.Freeze(now.Add(3 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := pack.StartRehearsal(now.Add(4 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := pack.Release(now.Add(5 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := pack.Touch(now.Add(6 * time.Minute)); ErrorCode(err) != "released_immutable" {
		t.Fatalf("发布后应为不可变，实际错误 %v", err)
	}
	if !errors.As(ErrVersionConflict, new(*RuleError)) {
		t.Fatal("并发冲突必须是可识别的领域错误")
	}
}

func TestCertificateDigestIsStable(t *testing.T) {
	now := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	pack, _ := NewTermPack("pack-1", "能源会议", "中文", "英语", now)
	first, _ := NewEntry("b", pack.ID, 1, "储能", "energy storage", "定义", "语境", "依据")
	second, _ := NewEntry("a", pack.ID, 1, "电网", "power grid", "定义", "语境", "依据")
	_ = first.Review(DecisionAccepted, "", "")
	_ = second.Review(DecisionAccepted, "", "")
	one, err := BuildCertificate("cert-1", "周岚", pack, []TermEntry{first, second}, now)
	if err != nil {
		t.Fatal(err)
	}
	two, err := BuildCertificate("cert-2", "周岚", pack, []TermEntry{second, first}, now)
	if err != nil {
		t.Fatal(err)
	}
	if one.ContentDigest != two.ContentDigest || string(one.SnapshotJSON) != string(two.SnapshotJSON) {
		t.Fatal("相同内容的摘要与快照必须不受输入顺序影响")
	}
}
