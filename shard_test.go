package shard

import (
	"fmt"
	"testing"
)

func TestNotCommittedAtStart(t *testing.T) {
	if New(4).Committed("slot1") {
		t.Fatal("初始不应处于已提交状态")
	}
}

func TestPutReturnsTrue(t *testing.T) {
	if !New(4).Put("slot1", 0, "abcd", "r1") {
		t.Fatal("正常分片应该写入成功")
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	if ErrBadSize == ErrIncomplete {
		t.Fatal("两种错误必须是不同的值")
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	if New(4).Scanned() != 0 {
		t.Fatal("Scanned 初始应该是 0")
	}
}

func TestPutRejectsBadSize(t *testing.T) {
	s := New(4)
	if s.Put("slot1", 0, "ab", "r1") {
		t.Fatal("首片长度不足 shardSize 必须拒绝")
	}
	if s.Put("slot1", 0, "abcde", "r2") {
		t.Fatal("超过 shardSize 的分片必须拒绝")
	}
	if s.Put("slot1", 0, "", "r3") {
		t.Fatal("空分片必须拒绝")
	}
	if err := s.Commit("slot1"); err != ErrIncomplete {
		t.Fatalf("非法分片不应落盘，Commit 应报不完整，got %v", err)
	}
}

func TestPutAcceptsShortFinalShard(t *testing.T) {
	s := New(4)
	if !s.Put("slot1", 0, "abcd", "r1") {
		t.Fatal("满片应写入成功")
	}
	if !s.Put("slot1", 1, "xy", "r2") {
		t.Fatal("短末片应写入成功")
	}
	if s.Put("slot1", 2, "zzzz", "r3") {
		t.Fatal("末片之后不应再写入")
	}
	if err := s.Commit("slot1"); err != nil {
		t.Fatalf("完整存档应提交成功，got %v", err)
	}
}

func TestPutShortShardNeedsFullPrefix(t *testing.T) {
	s := New(4)
	if s.Put("slot1", 2, "xy", "r1") {
		t.Fatal("前面缺片时短片不能算末片")
	}
}

func TestPutDuplicateReqID(t *testing.T) {
	s := New(4)
	if !s.Put("slot1", 0, "abcd", "r1") {
		t.Fatal("首次写入应成功")
	}
	if s.Put("slot1", 1, "xy", "r1") {
		t.Fatal("同一 reqID 重复调用必须返回 false")
	}
	if err := s.Commit("slot1"); err != ErrIncomplete {
		t.Fatalf("重复的末片不应落盘，Commit 应报不完整，got %v", err)
	}
	if !s.Put("slot1", 1, "xy", "r2") {
		t.Fatal("换新 reqID 的合法末片应写入成功")
	}
	if err := s.Commit("slot1"); err != nil {
		t.Fatalf("补齐后应提交成功，got %v", err)
	}
}

func TestCommitDetectsMissingShard(t *testing.T) {
	s := New(4)
	s.Put("slot1", 0, "abcd", "r1")
	s.Put("slot1", 2, "abcd", "r2")
	if err := s.Commit("slot1"); err != ErrIncomplete {
		t.Fatalf("缺第 1 片应报 ErrIncomplete，got %v", err)
	}
	if s.Committed("slot1") {
		t.Fatal("缺片时不应标记已提交")
	}
}

func TestCommitIsAtomic(t *testing.T) {
	s := New(4)
	s.Put("slot1", 0, "abcd", "r1")
	if err := s.Commit("slot1"); err != ErrIncomplete {
		t.Fatalf("没有末片应报 ErrIncomplete，got %v", err)
	}
	if s.Committed("slot1") {
		t.Fatal("提交失败不应改变提交状态")
	}
	if !s.Put("slot1", 1, "xy", "r2") {
		t.Fatal("补末片应写入成功")
	}
	if err := s.Commit("slot1"); err != nil {
		t.Fatalf("补齐后提交应成功，got %v", err)
	}
	if !s.Committed("slot1") {
		t.Fatal("提交成功后应处于已提交状态")
	}
}

func TestCommittedSaveRejectsWrites(t *testing.T) {
	s := New(4)
	s.Put("slot1", 0, "abcd", "r1")
	s.Put("slot1", 1, "xy", "r2")
	if err := s.Commit("slot1"); err != nil {
		t.Fatalf("完整存档应提交成功，got %v", err)
	}
	if s.Put("slot1", 2, "zzzz", "r3") {
		t.Fatal("已提交存档不应再写入")
	}
	if err := s.Commit("slot1"); err != nil {
		t.Fatalf("重复提交已提交存档应成功，got %v", err)
	}
	if !s.Committed("slot1") {
		t.Fatal("已提交状态不应改变")
	}
}

func TestCommitDoesNotScanAllSaves(t *testing.T) {
	s := New(4)
	for i := 0; i < 1000; i++ {
		s.Put(fmt.Sprintf("slot-%d", i), 0, "abcd", fmt.Sprintf("r-%d", i))
	}
	before := s.Scanned()
	s.Commit("slot-999")
	if got := s.Scanned() - before; got > 1 {
		t.Fatalf("单次 Commit 扫描量不应随存档数增长，got %d", got)
	}
}
