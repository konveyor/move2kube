/*
 *  Copyright IBM Corporation 2023
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package vcs

import (
	"errors"
	"sync/atomic"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
)

// ErrLimitExceeded is the error returned when the storage limit has been exceeded (usually during git clone)
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

// SetEncodedObject is a Storer interface method that is used to store an object
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

// Module is a Storer interface method that is used to get the working tree for a repo sub-module
func (s *Limited) Module(name string) (storage.Storer, error) {
	m, err := s.Storer.Module(name)
	if err != nil {
		return nil, err
	}
	n := s.N.Load()
	return Limit(m, n), nil
}
