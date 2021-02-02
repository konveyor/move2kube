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
	"fmt"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types"
	log "github.com/sirupsen/logrus"
)

// QACacheKind defines kind of QA Cache
const QACacheKind types.Kind = "QACache"

// Cache stores the answers for reuse
type Cache struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             CacheSpec `yaml:"spec,omitempty"`
}

// CacheSpec stores the cache data
type CacheSpec struct {
	file string `yaml:"-"`
	// Problems stores the list of problems with resolutions
	Problems []Problem `yaml:"solutions"`
}

// NewCache creates new cache instance
func NewCache(file string) (cache *Cache) {
	return &Cache{
		TypeMeta: types.TypeMeta{
			Kind:       string(QACacheKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
		Spec: CacheSpec{file: file},
	}
}

// Load loads and merges cache
func (cache *Cache) Load() error {
	c := Cache{}
	err := common.ReadMove2KubeYaml(cache.Spec.file, &c)
	if err != nil {
		log.Errorf("Unable to load cache : %s", err)
	} else {
		cache.merge(c)
		for i := range cache.Spec.Problems {
			cache.Spec.Problems[i].Resolved = true
		}
	}
	return err
}

// Write writes cache to disk
func (cache *Cache) Write() error {
	err := common.WriteYaml(cache.Spec.file, cache)
	if err != nil {
		log.Warnf("Unable to write cache : %s", err)
	}
	return err
}

// AddSolution adds a problem to solution cache
func (cache *Cache) AddSolution(p Problem) error {
	if p.Solution.Type == PasswordSolutionFormType {
		err := fmt.Errorf("Passwords are not added to the cache")
		log.Debug(err)
		return err
	}
	if !p.Resolved {
		err := fmt.Errorf("Unresolved problem. Not going to be added to cache")
		log.Warn(err)
		return err
	}
	added := false
	for i, cp := range cache.Spec.Problems {
		if cp.ID == p.ID {
			log.Debugf("A solution already exists in cache for [%s], rewriting", p.Desc)
			cache.Spec.Problems[i] = p
			added = true
			break
		}
	}
	if !added {
		cache.Spec.Problems = append(cache.Spec.Problems, p)
	}
	if err := cache.Write(); err != nil {
		log.Errorf("Failed to write to the cache file. Error: %q", err)
		return err
	}
	return nil
}

// GetSolution reads a solution for the problem
func (cache *Cache) GetSolution(p Problem) (ans Problem, err error) {
	if p.Resolved {
		log.Warnf("Problem already solved.")
		return p, nil
	}
	for _, cp := range cache.Spec.Problems {
		if (cp.ID == p.ID || cp.matches(p)) && cp.Resolved {
			err := p.SetAnswer(cp.Solution.Answer)
			return p, err
		}
	}
	return p, fmt.Errorf("The problem %+v was not found in the cache", p)
}

func (cache *Cache) merge(c Cache) {
	for _, p := range c.Spec.Problems {
		for _, op := range cache.Spec.Problems {
			if op.matches(p) {
				log.Warnf("There are two answers for %s in cache. Ignoring latter ones.", p.Desc)
				continue
			}
		}
		cache.Spec.Problems = append(cache.Spec.Problems, p)
	}
}
