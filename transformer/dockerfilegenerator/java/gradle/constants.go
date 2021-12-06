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

import "regexp"

const (
	repositoriesProp = "repositories"
	dependenciesProp = "dependencies"
	pluginsProp      = "plugins"
)

const (
	charTab              = '\t'
	charNewLine          = '\n'
	charCarriageReturn   = '\r'
	charSpace            = ' '
	charLeftParanthesis  = '('
	charRightParanthesis = ')'
	charPeriod           = '.'
	charSlash            = '/'
	charEquals           = '='
	charArrayStart       = '['
	charArrayEnd         = ']'
	charBlockStart       = '{'
	charBlockEnd         = '}'

	keywordDef = "def"
	keywordIf  = "if"

	singleLineCommentStart = `//`
	blockCommentStart      = `/*`
	blockCommentEnd        = `*/`
)

var (
	depsKeywordString      = `(?m)\s*([A-Za-z0-9_-]+)\s*`
	depsKeywordStringRegex = regexp.MustCompile(depsKeywordString)
	depsEasyGavStringRegex = regexp.MustCompile(`(["']?)([\w.-]+):([\w.-]+):([\w\[\]\(\),+.-]+)\1`)
	depsHardGavStringRegex = regexp.MustCompile(depsKeywordString + `(?:\((.*)\)|(.*))`)
	depsItemBlockRegex     = regexp.MustCompile(depsKeywordString + `(?m)\(((["']?)(.*)\3)\)\s*\{`)
	depsExcludeLineRegex   = regexp.MustCompile(`exclude[ \\t]+([^\\n]+)`)
	pluginsLinePattern     = regexp.MustCompile(`(?m)(id|version)(?:\s)(["']?)([A-Za-z0-9.]+)\2`)

	projectRegex  = regexp.MustCompile(`(project\([^\)]+\))`)
	functionRegex = regexp.MustCompile(`\w+\(.*\);?$`)

	whitespacecharacters = map[rune]bool{
		charTab:            true,
		charNewLine:        true,
		charCarriageReturn: true,
		charSpace:          true,
	}
)
