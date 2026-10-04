package shard

import "testing"

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
