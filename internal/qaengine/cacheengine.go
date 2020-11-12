/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package qaengine

import (
	qatypes "github.com/konveyor/move2kube/types/qaengine"
)

// CacheEngine handles cache
type CacheEngine struct {
	cache qatypes.Cache
}

// NewCacheEngine creates a new cache instance
func NewCacheEngine(cf string) *CacheEngine {
	ce := new(CacheEngine)
	ce.cache = qatypes.NewCache(cf)
	return ce
}

// StartEngine starts the cache engine
func (c *CacheEngine) StartEngine() error {
	return c.cache.Load()
}

// FetchAnswer fetches the answer using cache
func (c *CacheEngine) FetchAnswer(prob qatypes.Problem) (ans qatypes.Problem, err error) {
	return c.cache.GetSolution(prob)
}
