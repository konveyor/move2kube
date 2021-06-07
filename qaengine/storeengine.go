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

// StoreEngine handles cache
type StoreEngine struct {
	store qatypes.Store
}

// StartEngine loads the config from the store
func (se *StoreEngine) StartEngine() error {
	return se.store.Load()
}

// FetchAnswer fetches the answer from the store
func (se *StoreEngine) FetchAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	return se.store.GetSolution(prob)
}

// IsInteractiveEngine returns true if the engine interacts with the user
func (*StoreEngine) IsInteractiveEngine() bool {
	return false
}

// NewStoreEngineFromCache creates a new cache instance
func NewStoreEngineFromCache(cacheFile string) *StoreEngine {
	return &StoreEngine{store: qatypes.NewCache(cacheFile)}
}
