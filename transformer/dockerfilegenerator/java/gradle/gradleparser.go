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
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Inspired from https://github.com/ninetwozero/gradle-to-js/blob/master/lib/parser.js

// ParseGardleBuildFile parses a gradle build file
func ParseGardleBuildFile(buildFilePath string) (gradleBuild Gradle, err error) {
	state := gradleParseState{}
	buildFile, err := os.ReadFile(buildFilePath)
	if err != nil {
		logrus.Errorf("Unable to read gradle build file : %s", err)
		return Gradle{}, err
	}
	return deepParse(string(buildFile), &state, false, true), nil
}

func deepParse(chunk string, state *gradleParseState, keepFunctionCalls, skipEmptyValues bool) (parsedGradleOutput Gradle) {
	parsedGradleOutput = Gradle{}
	var character rune
	var tempString, commentText string
	var currentKey string
	parsingKey := true
	isBeginningOfLine := true

	for chunkLength := len([]rune(chunk)); state.index < chunkLength; state.index++ {
		character = ([]rune(chunk))[state.index]
		if isBeginningOfLine && isWhitespace(character) {
			continue
		}

		if !state.comment.parsing && isBeginningOfLine && isStartOfComment(tempString) {
			isBeginningOfLine = false
			if isSingleLineComment(tempString) {
				state.comment.setSingleLine()
			} else {
				state.comment.setMultiLine()
			}
			continue
		}

		if state.comment.multiLine && isEndOfMultiLineComment(commentText) {
			state.comment.reset()
			isBeginningOfLine = true
			tempString = ""
			commentText = ""
			continue
		}

		if state.comment.parsing && character != charNewLine {
			commentText += string(character)
			continue
		}

		if state.comment.parsing && isLineBreakCharacter(character) {
			if state.comment.singleLine {
				state.comment.reset()
				isBeginningOfLine = true

				currentKey = ""
				tempString = ""
				commentText = ""
			}
			continue
		}

		if parsingKey && !keepFunctionCalls && character == charLeftParanthesis {
			skipFunctionCall(chunk, state)
			currentKey = ""
			tempString = ""
			isBeginningOfLine = true
			continue
		}
		if isLineBreakCharacter(character) {
			if currentKey == "" && tempString != "" {
				if parsingKey {
					if isFunctionCall(tempString) && !keepFunctionCalls {
						continue
					} else {
						currentKey = strings.TrimSpace(tempString)
						tempString = ""
					}
				}
			}
			if tempString != "" || (currentKey != "" && !skipEmptyValues) {
				addValueToStructure(&parsedGradleOutput, currentKey, trimWrappingQuotes(tempString))
				currentKey = ""
				tempString = ""
			}
			parsingKey = true
			isBeginningOfLine = true

			state.comment.reset()
			continue
		}
		// Only parse as an array if the first *real* char is a [
		if !parsingKey && tempString == "" && character == charArrayStart {
			addValueToStructure(&parsedGradleOutput, currentKey, parseArray(chunk, state)...)
			currentKey = ""
			tempString = ""
			continue
		}
		if character == charBlockStart {
			// We need to skip the current (=start) character so that we literally "step into" the next closure/block
			state.index++
			switch currentKey {
			case repositoriesProp:
				parsedGradleOutput.Repositories = append(parsedGradleOutput.Repositories, parseRepositoryClosure(chunk, state)...)
			case dependenciesProp:
				parsedGradleOutput.Dependencies = append(parsedGradleOutput.Dependencies, parseDependencyClosure(chunk, state)...)
			case pluginsProp:
				parsedGradleOutput.Plugins = append(parsedGradleOutput.Plugins, parsePluginsClosure(chunk, state)...)
			default:
				if parsedGradleOutput.Blocks == nil {
					parsedGradleOutput.Blocks = map[string]Gradle{}
				}
				if _, ok := parsedGradleOutput.Blocks[currentKey]; ok {
					gb := parsedGradleOutput.Blocks[currentKey]
					gb.Merge(deepParse(chunk, state, keepFunctionCalls, skipEmptyValues))
					parsedGradleOutput.Blocks[currentKey] = gb
				} else {
					parsedGradleOutput.Blocks[currentKey] = deepParse(chunk, state, keepFunctionCalls, skipEmptyValues)
				}
			}
			currentKey = ""
		} else if character == charBlockEnd {
			currentKey = ""
			tempString = ""
			break
		} else if isDelimiter(character) && parsingKey {
			if isKeyword(tempString) {
				if tempString == keywordDef {
					tempString = fetchDefinedNameOrSkipFunctionDefinition(chunk, state)
				} else if tempString == keywordIf {
					skipIfStatement(chunk, state)
					currentKey = ""
					tempString = ""
					continue
				}
			}
			currentKey = tempString
			tempString = ""
			parsingKey = false
			if currentKey == "" {
				continue
			}
		} else {
			if tempString == "" && isDelimiter(character) {
				continue
			}
			tempString += string(character)
			isBeginningOfLine = isBeginningOfLine && (character == charSlash || isStartOfComment(tempString))
		}
	}
	// Add the last value to the structure
	addValueToStructure(&parsedGradleOutput, currentKey, trimWrappingQuotes(tempString))
	return parsedGradleOutput
}

func skipIfStatement(chunk string, state *gradleParseState) bool {
	skipFunctionCall(chunk, state)
	chunkAsRune := []rune(chunk)
	var character rune
	var hasFoundTheCurlyBraces, hasFoundAStatementWithoutBraces bool
	curlyBraceCount := 0
	for max := len(chunkAsRune); state.index < max; state.index++ {
		character = chunkAsRune[state.index]
		if hasFoundAStatementWithoutBraces {
			if isLineBreakCharacter(character) {
				break
			}
		} else {
			if character == charBlockStart {
				hasFoundTheCurlyBraces = true
				curlyBraceCount++
			} else if character == charBlockEnd {
				curlyBraceCount--
			} else if !hasFoundTheCurlyBraces && !isWhitespace(character) {
				hasFoundAStatementWithoutBraces = true
			}
			if hasFoundTheCurlyBraces && curlyBraceCount == 0 {
				break
			}
		}
	}
	return curlyBraceCount == 0
}

func skipFunctionDefinition(chunk string, state *gradleParseState) {
	parenthesisNest := 1
	state.index++
	chunkAsRuneA := []rune(chunk)
	chunkLen := len(chunkAsRuneA)
	var character rune
	for chunkLen < state.index && parenthesisNest != 0 {
		character = chunkAsRuneA[state.index]
		if character == charLeftParanthesis {
			parenthesisNest++
		} else if character == charRightParanthesis {
			parenthesisNest--
		}
		state.index++
		character = chunkAsRuneA[state.index]
	}
	for chunkLen < state.index && character != charBlockStart {
		state.index++
		character = chunkAsRuneA[state.index]
	}
	state.index++
	character = chunkAsRuneA[state.index]
	blockNest := 1
	for chunkLen < state.index && blockNest != 0 {
		if character == charBlockStart {
			blockNest++
		} else if character == charBlockEnd {
			blockNest--
		}
		state.index++
		character = chunkAsRuneA[state.index]
	}
	state.index--
}

func parseDependencyClosure(chunk string, state *gradleParseState) []GradleDependency {
	specialClosures := parseSpecialClosure(chunk, state)
	gradleDependencies := []GradleDependency{}
	for _, specialClosure := range specialClosures {
		gradleDependencies = append(gradleDependencies, createStructureForDependencyItem(specialClosure))
	}
	return gradleDependencies
}

func createStructureForDependencyItem(data string) GradleDependency {
	gdi := GradleDependency{}
	if match := depsItemBlockRegex.FindStringSubmatch(data); len(match) > 2 {
		excludes := []map[string]string{}
		excludeMatches := depsExcludeLineRegex.FindAllStringSubmatch(data, -1)
		for _, excludeMatch := range excludeMatches {
			excludes = append(excludes, parseMapNotation(excludeMatch[0][findFirstSpaceOrTabPosition(excludeMatch[0]):]))
		}
		gdi.GradleGAV = parseGavString(match[2])
		gdi.Type = match[1]
		gdi.Excludes = excludes
	} else {
		gdi.GradleGAV = parseGavString(data)
		parsed := depsKeywordStringRegex.FindStringSubmatch(data)
		if len(parsed) > 1 {
			gdi.Type = parsed[1]
		}
	}
	return gdi
}

func parsePluginsClosure(chunk string, state *gradleParseState) []GradlePlugin {
	specialClosures := parseSpecialClosure(chunk, state)
	gradlePlugins := []GradlePlugin{}
	for _, specialClosure := range specialClosures {
		gradlePlugins = append(gradlePlugins, createStructureForPlugin(specialClosure))
	}
	return gradlePlugins
}

func createStructureForPlugin(pluginRow string) map[string]string {
	plugin := map[string]string{}
	matches := pluginsLineRegex.FindAllStringSubmatch(pluginRow)
	for _, match := range matches {
		if len(match) > 1 {
			plugin[match[1]] = match[3]
		}
	}
	return plugin
}

func findFirstSpaceOrTabPosition(input string) int {
	position := strings.Index(input, " ")
	if position == -1 {
		position = strings.Index(input, "\t")
	}
	return position
}

func parseGavString(gavString string) (gav GradleGAV) {
	gav = GradleGAV{}
	easyGavStringMatches := depsEasyGavStringRegex.FindStringSubmatch(gavString)
	if len(easyGavStringMatches) > 3 {
		gav.Group = easyGavStringMatches[2]
		gav.Name = easyGavStringMatches[3]
		gav.Version = easyGavStringMatches[4]
	} else if strings.Contains(gavString, `project(`) {
		gav.Name = projectRegex.FindString(gavString)
	} else {
		hardGavMatches := depsHardGavStringRegex.FindStringSubmatch(gavString)
		if len(hardGavMatches) > 2 {
			if hardGavMatches[3] != "" {
				gav = parseMapNotationWithFallback(gav, hardGavMatches[3], "")
			} else {
				gav = parseMapNotationWithFallback(gav, hardGavMatches[2], "")
			}
		} else {
			gav = parseMapNotationWithFallback(gav, gavString, gavString[findFirstSpaceOrTabPosition(gavString):])
		}
	}
	return gav
}

func parseMapNotationWithFallback(gav GradleGAV, str, name string) GradleGAV {
	parsedMap := parseMapNotation(str)
	if _, ok := parsedMap["name"]; ok {
		return GradleGAV{
			Group:   parsedMap["group"],
			Name:    parsedMap["name"],
			Version: parsedMap["version"],
		}
	}
	if name == "" {
		name = str
	}
	gav.Name = name
	return gav
}

func parseMapNotation(input string) (parsedMap map[string]string) {
	parsedMap = map[string]string{}
	currentKey := ""
	var quotation rune
	inputAsRune := []rune(input)
	for i, max := 0, len(inputAsRune); i < max; i++ {
		if inputAsRune[i] == ':' {
			currentKey = strings.TrimSpace(currentKey)
			parsedMap[currentKey] = ""
			var innerLoop rune
			for i = i + 1; i < max; i++ {
				if innerLoop == 0 {
					// Skip any leading spaces before the actual value
					if isWhitespace(inputAsRune[i]) {
						continue
					}
				}
				// We just take note of what the "latest" quote was so that we can
				if inputAsRune[i] == '"' || inputAsRune[i] == '\'' {
					quotation = inputAsRune[i]
					continue
				}
				// Moving on to the next value if we find a comma
				if inputAsRune[i] == ',' {
					parsedMap[currentKey] = strings.TrimSpace(parsedMap[currentKey])
					currentKey = ""
					break
				}
				parsedMap[currentKey] += string(inputAsRune[i])
				innerLoop++
			}
		} else {
			currentKey += string(inputAsRune[i])
		}
	}
	// If the last character contains a quotation mark, we remove it
	if val, ok := parsedMap[currentKey]; ok {
		parsedMap[currentKey] = strings.TrimSuffix(strings.TrimSpace(val), string(quotation))
	}
	return parsedMap
}

func parseRepositoryClosure(chunk string, state *gradleParseState) (repositories []GradleRepository) {
	repositories = []GradleRepository{}
	parsedRepos := deepParse(chunk, state, true, false)
	for parsedRepoType, value := range parsedRepos.Metadata {
		if len(value) > 0 && value[0] != "" {
			repositories = append(repositories, GradleRepository{Type: parsedRepoType, Data: GradleRepositoryData{
				Name: value[0],
			}})
		} else {
			repositories = append(repositories, GradleRepository{Type: "unknown", Data: GradleRepositoryData{Name: parsedRepoType}})
		}
	}
	return repositories
}

func parseSpecialClosure(chunk string, state *gradleParseState) (closures []string) {
	closures = []string{}
	// openBlockCount starts at 1 due to us entering after "<key> {"
	openBlockCount := 1
	currentKey := ""
	currentValue := ""
	chunkAsRune := []rune(chunk)

	isInItemBlock := false
	for ; state.index < len(chunkAsRune); state.index++ {
		if chunkAsRune[state.index] == charBlockStart {
			openBlockCount++
		} else if chunkAsRune[state.index] == charBlockEnd {
			openBlockCount--
		} else {
			currentKey += string(chunkAsRune[state.index])
		}

		// Keys shouldn't have any leading nor trailing whitespace
		currentKey = strings.TrimSpace(currentKey)

		if isStartOfComment(currentKey) {
			var commentText = currentKey
			for state.index++; state.index < len(chunkAsRune); state.index++ {
				if isCommentComplete(commentText, chunkAsRune[state.index]) {
					currentKey = ""
					break
				}
				commentText += string(chunkAsRune[state.index])
			}
		}

		if currentKey != "" && isWhitespace(chunkAsRune[state.index]) {
			var character rune
			for state.index++; state.index < len(chunkAsRune); state.index++ {
				character = chunkAsRune[state.index]
				currentValue += string(character)
				if character == charBlockStart {
					isInItemBlock = true
				} else if isInItemBlock && character == charBlockEnd {
					isInItemBlock = false
				} else if !isInItemBlock {
					if isLineBreakCharacter(character) && currentValue != "" {
						break
					}
				}
			}

			closures = append(closures, currentKey+" "+currentValue)
			currentKey = ""
			currentValue = ""
		}

		if openBlockCount == 0 {
			break
		}
	}
	return closures
}

func fetchDefinedNameOrSkipFunctionDefinition(chunk string, state *gradleParseState) string {
	var character rune
	checkAsRune := []rune(chunk)
	temp := ""
	isVariableDefinition := true
	for max := len(checkAsRune); state.index < max; state.index++ {
		character = checkAsRune[state.index]

		if character == charEquals {
			// Variable definition, break and return name
			break
		} else if character == charLeftParanthesis {
			// Function definition, skip parsing
			isVariableDefinition = false
			skipFunctionDefinition(chunk, state)
			break
		}
		temp += string(character)
	}

	if isVariableDefinition {
		values := strings.Split(strings.TrimSpace(temp), " ")
		return values[len(values)-1]
	}
	return ""
}

func parseArray(chunk string, state *gradleParseState) []string {
	var character rune
	chunkAsRune := []rune(chunk)
	temp := ""
	for max := len(chunkAsRune); state.index < max; state.index++ {
		character = chunkAsRune[state.index]
		if character == charArrayStart {
			continue
		} else if character == charArrayEnd {
			break
		}
		temp += string(character)
	}
	elems := strings.Split(temp, ",")
	for elemI, elem := range elems {
		elems[elemI] = trimWrappingQuotes(strings.TrimSpace(elem))
	}
	return elems
}

func skipFunctionCall(chunk string, state *gradleParseState) bool {
	openParenthesisCount := 0
	checkAsRune := []rune(chunk)
	var character rune
	for max := len(checkAsRune); state.index < max; state.index++ {
		character = checkAsRune[state.index]
		if character == charLeftParanthesis {
			openParenthesisCount++
		} else if character == charRightParanthesis {
			openParenthesisCount--
		}
		if openParenthesisCount == 0 && !isWhitespace(character) {
			state.index++
			break
		}
	}
	return openParenthesisCount == 0
}

func addValueToStructure(gradleBuild *Gradle, currentKey string, value ...string) {
	switch currentKey {
	case "":
		return
	case repositoriesProp:
		fallthrough
	case dependenciesProp:
		fallthrough
	case pluginsProp:
		logrus.Errorf("Incompatible value while parsing for %s", currentKey)
	default:
		if gradleBuild.Metadata == nil {
			gradleBuild.Metadata = map[string][]string{}
		}
		gradleBuild.Metadata[currentKey] = append(gradleBuild.Metadata[currentKey], value...)
	}
}

func trimWrappingQuotes(str string) string {
	doubleQuote := `"`
	singleQuote := "'"
	str = strings.TrimSpace(str)
	if strings.HasPrefix(str, doubleQuote) {
		str = strings.TrimPrefix(strings.TrimSuffix(str, doubleQuote), doubleQuote)
	} else if strings.HasPrefix(str, singleQuote) {
		str = strings.TrimPrefix(strings.TrimSuffix(str, singleQuote), singleQuote)
	}
	return str
}

func isDelimiter(character rune) bool {
	return character == charSpace || character == charEquals
}

func isWhitespace(character rune) bool {
	return whitespacecharacters[character]
}

func isLineBreakCharacter(character rune) bool {
	return character == charCarriageReturn || character == charNewLine
}

func isKeyword(str string) bool {
	return str == keywordDef || str == keywordIf
}

func isSingleLineComment(comment string) bool {
	return comment[0:2] == singleLineCommentStart
}

func isStartOfComment(snippet string) bool {
	return snippet == blockCommentStart || snippet == singleLineCommentStart
}

func isCommentComplete(text string, next rune) bool {
	return (isLineBreakCharacter(next) && isSingleLineComment(text)) || (isWhitespace(next) && isEndOfMultiLineComment(text))
}

func isEndOfMultiLineComment(comment string) bool {
	if len(comment) < 2 {
		return false
	}
	return comment[len(comment)-2:] == blockCommentEnd
}

func isFunctionCall(str string) bool {
	return functionRegex.MatchString(str)
}

// GetDirNameFromDir returns the directory name from a dir obj
func GetDirNameFromDir(str string) string {
	str = trimWrappingQuotes(strings.TrimSpace(strings.Trim(strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(str), "layout.buildDirectory.dir")), "("), ")")))
	return str
}
