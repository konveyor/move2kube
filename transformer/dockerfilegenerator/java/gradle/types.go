/*
 *  Copyright IBM Corporation 2021
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

package main

type GradleBuild struct {
	Repositories []GradleRepository     `yaml:"repositories" json:"repositories"`
	Dependencies []GradleDependency     `yaml:"dependencies" json:"dependencies"`
	Plugins      []GradlePlugin         `yaml:"plugins" json:"plugins"`
	Metadata     map[string][]string    `yaml:"metadata" json:"metadata"`
	Blocks       map[string]GradleBuild `yaml:"blocks" json:"blocks"`
}

func (g *GradleBuild) Merge(newg GradleBuild) {
	g.Repositories = append(g.Repositories, newg.Repositories...)
	g.Dependencies = append(g.Dependencies, newg.Dependencies...)
	g.Plugins = append(g.Plugins, newg.Plugins...)
	for mi, m := range newg.Metadata {
		g.Metadata[mi] = append(g.Metadata[mi], m...)
	}
	for bi, b := range newg.Blocks {
		if ob, ok := g.Blocks[bi]; ok {
			ob.Merge(b)
			g.Blocks[bi] = ob
		} else {
			g.Blocks[bi] = b
		}
	}
}

type GradleGAV struct {
	Group   string `yaml:"group" json:"group"`
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
}

type GradleDependency struct {
	GradleGAV
	Type     string              `yaml:"type" json:"type"`
	Excludes []map[string]string `yaml:"excludes" json:"excludes"`
}

type GradleRepository struct {
	Type string               `yaml:"type" json:"type"`
	Data GradleRepositoryData `yaml:"data" json:"data"`
}

type GradleRepositoryData struct {
	Name string `yaml:"name" json:"name"`
}

type GradlePlugin map[string]string

type gradleParseState struct {
	index   int
	comment gradleComment
}

type gradleComment struct {
	parsing, singleLine, multiLine bool
}

func (g *gradleComment) setSingleLine() {
	g.setCommentState(true, false)
}

func (g *gradleComment) setMultiLine() {
	g.setCommentState(false, true)
}

func (g *gradleComment) reset() {
	g.setCommentState(false, false)
}

func (g *gradleComment) setCommentState(singleLine, multiLine bool) {
	g.singleLine = singleLine
	g.multiLine = multiLine
	g.parsing = singleLine || multiLine
}