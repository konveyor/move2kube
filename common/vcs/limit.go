package vcs

import (
	"errors"
	"sync/atomic"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
)

var ErrLimitExceeded = errors.New("repo size limit exceeded")

// Limited wraps git.Storer to limit the number of bytes that can be stored.
type Limited struct {
	storage.Storer
	N atomic.Int64
}

// Limit returns a git.Storer limited to the specified number of bytes.
func Limit(s storage.Storer, n int64) storage.Storer {
	if n < 0 {
		return s
	}
	l := &Limited{Storer: s}
	l.N.Store(n)
	return l
}

func (s *Limited) SetEncodedObject(obj plumbing.EncodedObject) (plumbing.Hash, error) {
	objSize := obj.Size()
	n := s.N.Load()
	if n-objSize < 0 {
		return plumbing.ZeroHash, ErrLimitExceeded
	}
	for !s.N.CompareAndSwap(n, n-objSize) {
		n = s.N.Load()
		if n-objSize < 0 {
			return plumbing.ZeroHash, ErrLimitExceeded
		}
	}
	return s.Storer.SetEncodedObject(obj)
}

func (s *Limited) Module(name string) (storage.Storer, error) {
	m, err := s.Storer.Module(name)
	if err != nil {
		return nil, err
	}
	n := s.N.Load()
	return Limit(m, n), nil
}
