package shard

import (
	"errors"
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
		t.Fatal("长度不足 shardSize 的第 0 片不应写入")
	}
	if s.Put("slot1", 0, "abcdef", "r2") {
		t.Fatal("长度超过 shardSize 的分片不应写入")
	}
	if s.Put("slot1", 0, "", "r3") {
		t.Fatal("空分片不应写入")
	}
	if err := s.Commit("slot1"); !errors.Is(err, ErrIncomplete) {
		t.Fatal("写入失败不应留下分片，提交应报不完整")
	}
}

func TestPutRejectsShortShardAfterFinal(t *testing.T) {
	s := New(4)
	if !s.Put("slot1", 0, "abcd", "r1") || !s.Put("slot1", 1, "xy", "r2") {
		t.Fatal("满片加末片应该写入成功")
	}
	if s.Put("slot1", 2, "efgh", "r3") {
		t.Fatal("末片之后不应再写入分片")
	}
}

func TestPutIdempotentReqID(t *testing.T) {
	s := New(4)
	if !s.Put("slot1", 0, "abcd", "same") {
		t.Fatal("首次写入应该成功")
	}
	if s.Put("slot1", 1, "efgh", "same") {
		t.Fatal("同一个 reqID 重复调用应返回 false")
	}
	if !s.Put("slot1", 1, "efgh", "r2") {
		t.Fatal("重复调用不应真的写入，换个 reqID 补写应成功")
	}
}

func TestCommitDetectsMissingShard(t *testing.T) {
	s := New(4)
	s.Put("slot1", 0, "abcd", "r1")
	s.Put("slot1", 2, "xy", "r2")
	if err := s.Commit("slot1"); !errors.Is(err, ErrIncomplete) {
		t.Fatal("缺第 1 片应返回 ErrIncomplete")
	}
	if s.Committed("slot1") {
		t.Fatal("缺片时不应标记已提交")
	}
}

func TestCommitRequiresFinalShard(t *testing.T) {
	s := New(4)
	s.Put("slot1", 0, "abcd", "r1")
	if err := s.Commit("slot1"); !errors.Is(err, ErrIncomplete) {
		t.Fatal("只有满片、末片未就位应报不完整")
	}
	if s.Committed("slot1") {
		t.Fatal("提交失败不应改变提交状态")
	}
}

func TestCommitAtomicAndImmutable(t *testing.T) {
	s := New(4)
	s.Put("slot1", 0, "abcd", "r1")
	if err := s.Commit("slot1"); !errors.Is(err, ErrIncomplete) {
		t.Fatal("缺末片应提交失败")
	}
	if s.Committed("slot1") {
		t.Fatal("失败的提交不应标记已提交")
	}
	if !s.Put("slot1", 1, "xy", "r2") {
		t.Fatal("补齐末片应该写入成功")
	}
	if err := s.Commit("slot1"); err != nil {
		t.Fatal("分片齐全后应提交成功")
	}
	if !s.Committed("slot1") {
		t.Fatal("提交成功后应处于已提交状态")
	}
	if s.Put("slot1", 2, "zz", "r3") {
		t.Fatal("已提交的存档不应再被写入改变")
	}
	if err := s.Commit("slot1"); err != nil {
		t.Fatal("重复提交已提交的存档应直接成功")
	}
}

func TestCommitScanDoesNotGrowWithSaves(t *testing.T) {
	s := New(4)
	for i := 0; i < 1000; i++ {
		s.Put(fmt.Sprintf("slot-%d", i), 0, "abcd", fmt.Sprintf("r-%d", i))
	}
	before := s.Scanned()
	for i := 0; i < 1000; i++ {
		s.Commit("slot-999")
	}
	if got := s.Scanned() - before; got > 1000 {
		t.Fatalf("Commit 扫描量不应随存档数线性增长: %d", got)
	}
}
