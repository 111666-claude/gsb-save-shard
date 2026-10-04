// Package shard 是存档分片：分片大小校验、写入幂等、缺片检测与原子提交。
package shard

import "errors"

// ErrBadSize 分片长度不合法。
var ErrBadSize = errors.New("分片长度不合法")

// ErrIncomplete 分片不完整。
var ErrIncomplete = errors.New("分片不完整")

// Store 是分片存档台账。
type Store struct {
	shardSize int
	saves     map[string]map[int]string
	memo      map[string]bool
	committed map[string]bool
	scanned   int
}

// New 建台账。
func New(shardSize int) *Store {
	return &Store{shardSize: shardSize, saves: map[string]map[int]string{},
		memo: map[string]bool{}, committed: map[string]bool{}}
}

// Put 写入一个分片。
// 除最后一片外，每个分片长度必须等于 shardSize；短片即末片，
// 仅在 0..index-1 连续齐全且后面没有分片时合法，其余一律返回 false 且不写入。
// 同一个 reqID 只写入一次，重复调用返回 false；已提交的存档拒绝再写入。
func (s *Store) Put(save string, index int, data, reqID string) bool {
	s.scanned++
	if index < 0 || s.memo[reqID] || s.committed[save] {
		return false
	}
	shards := s.saves[save]
	if _, dup := shards[index]; dup {
		return false
	}
	last := -1
	for i := range shards {
		if i > last {
			last = i
		}
	}
	if last >= 0 && len(shards[last]) < s.shardSize {
		return false
	}
	switch size := len(data); {
	case size == 0 || size > s.shardSize:
		return false
	case size < s.shardSize:
		if index == 0 || index <= last {
			return false
		}
		for i := 0; i < index; i++ {
			if _, ok := shards[i]; !ok {
				return false
			}
		}
	}
	if shards == nil {
		shards = map[int]string{}
		s.saves[save] = shards
	}
	shards[index] = data
	s.memo[reqID] = true
	return true
}

// Commit 校验并提交。分片下标须从 0 连续无缺且以短末片收尾，
// 否则返回 ErrIncomplete；只有校验通过才标记已提交，失败时不改变提交状态。
// 只扫描该存档自身的分片，开销不随存档总数增长。
func (s *Store) Commit(save string) error {
	shards := s.saves[save]
	s.scanned += len(shards)
	if len(shards) == 0 {
		return ErrIncomplete
	}
	last := -1
	for i := range shards {
		if i > last {
			last = i
		}
	}
	for i := 0; i <= last; i++ {
		data, ok := shards[i]
		if !ok || (i == last && len(data) >= s.shardSize) {
			return ErrIncomplete
		}
	}
	s.committed[save] = true
	return nil
}

// Committed 判断是否已提交。
func (s *Store) Committed(save string) bool { return s.committed[save] }

// Scanned 是累计扫描量（规模观测）。
func (s *Store) Scanned() int { return s.scanned }
