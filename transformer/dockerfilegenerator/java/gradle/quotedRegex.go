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

package gradle

import (
	"regexp"
	"strings"
)

type quotedRegex struct {
	regexes []regexp.Regexp
}

func (q *quotedRegex) Init(regex string) {
	q.regexes = []regexp.Regexp{}
	q.regexes = append(q.regexes, *regexp.MustCompile(regex))
	q.regexes = append(q.regexes, *regexp.MustCompile(strings.ReplaceAll(regex, `"`, "'")))
}

func (q *quotedRegex) FindStringSubmatch(str string) []string {
	var regex regexp.Regexp
	matchIndex := -1
	for _, r := range q.regexes {
		match := r.FindStringSubmatchIndex(str)
		if len(match) > 0 {
			if matchIndex == -1 || matchIndex > match[0] {
				matchIndex = match[0]
				regex = r
			}
		}
	}
	if matchIndex != -1 {
		return regex.FindStringSubmatch(str)
	}
	return nil
}

func (q *quotedRegex) FindAllStringSubmatch(str string) (matches [][]string) {
	matches = [][]string{}
	for _, r := range q.regexes {
		matches = append(matches, r.FindAllStringSubmatch(str, -1)...)
	}
	return matches
}
