// Package shard 是存档分片：分片大小校验、写入幂等、缺片检测与原子提交。
package shard

import "errors"

// ErrBadSize 分片长度不合法。
var ErrBadSize = errors.New("分片长度不合法")

// ErrIncomplete 分片不完整。
var ErrIncomplete = errors.New("分片不完整")

// saveState 是单个存档的分片台账。
type saveState struct {
	shards     map[int]string
	maxIndex   int  // 已写入的最大下标，-1 表示空
	finalIndex int  // 末片（长度不足 shardSize）下标，-1 表示尚未写入末片
	committed  bool // 是否已提交
}

// Store 是分片存档台账。
type Store struct {
	shardSize int
	saves     map[string]*saveState
	seen      map[string]bool // 已消费的 reqID
	scanned   int
}

// New 建台账。
func New(shardSize int) *Store {
	return &Store{shardSize: shardSize, saves: map[string]*saveState{}, seen: map[string]bool{}}
}

// Put 写入一个分片。除最后一片外每个分片长度必须等于 shardSize；
// 同一个 reqID 只写入一次；已提交的存档不再改变。不合法时不写入并返回 false。
func (s *Store) Put(save string, index int, data, reqID string) bool {
	s.scanned++
	if s.seen[reqID] {
		return false
	}
	st := s.saves[save]
	if st == nil {
		st = &saveState{shards: map[int]string{}, maxIndex: -1, finalIndex: -1}
	}
	if st.committed || !s.validShard(st, index, len(data)) {
		return false
	}
	st.shards[index] = data
	if len(data) < s.shardSize {
		st.finalIndex = index
	} else if st.finalIndex == index {
		st.finalIndex = -1
	}
	if index > st.maxIndex {
		st.maxIndex = index
	}
	s.saves[save] = st
	s.seen[reqID] = true
	return true
}

// validShard 校验分片长度与位置：满片长度必须等于 shardSize，且不能写到
// 已有末片之后；不足 shardSize 的是末片，不能是第 0 片，其后不能已有分片，
// 一个存档也只能有一个末片。
func (s *Store) validShard(st *saveState, index, size int) bool {
	if index < 0 || size == 0 || size > s.shardSize {
		return false
	}
	if size == s.shardSize {
		return st.finalIndex < 0 || index <= st.finalIndex
	}
	return index > 0 && index >= st.maxIndex && (st.finalIndex < 0 || st.finalIndex == index)
}

// Commit 校验并提交：分片下标从 0 连续无缺且末片就位才标记已提交；
// 校验失败返回 ErrIncomplete 且不改变提交状态；重复提交已提交的存档直接成功。
func (s *Store) Commit(save string) error {
	st := s.saves[save]
	if st == nil || len(st.shards) == 0 {
		return ErrIncomplete
	}
	if st.committed {
		return nil
	}
	s.scanned += len(st.shards)
	if len(st.shards) != st.maxIndex+1 || st.finalIndex != st.maxIndex {
		return ErrIncomplete
	}
	st.committed = true
	return nil
}

// Committed 判断是否已提交。
func (s *Store) Committed(save string) bool {
	st := s.saves[save]
	return st != nil && st.committed
}

// Scanned 是累计扫描量（规模观测）。
func (s *Store) Scanned() int { return s.scanned }
