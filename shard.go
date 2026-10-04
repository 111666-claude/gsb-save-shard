// Package shard 是存档分片：分片大小校验、写入幂等、缺片检测与原子提交。
// 缺陷：不校验分片大小、同一请求重复写入、缺片也提交成功、
// 提交失败仍标记已提交、每次校验扫描全部存档。
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
func (s *Store) Put(save string, index int, data, reqID string) bool {
	s.scanned += len(s.saves)
	if s.saves[save] == nil {
		s.saves[save] = map[int]string{}
	}
	s.saves[save][index] = data
	s.memo[reqID] = true
	return true
}

// Commit 校验并提交，返回缺失的分片数。
func (s *Store) Commit(save string) error {
	s.scanned += len(s.saves)
	s.committed[save] = true
	return nil
}

// Committed 判断是否已提交。
func (s *Store) Committed(save string) bool { return s.committed[save] }

// Scanned 是累计扫描量（规模观测）。
func (s *Store) Scanned() int { return s.scanned }
