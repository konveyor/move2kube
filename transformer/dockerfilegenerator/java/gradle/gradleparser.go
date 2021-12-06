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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

type GradleBuild struct {

}

type GradleGav struct {
	Group string `yaml:"group" json:"group"`
	Name string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
}

type GradleDependencyItem struct {
	GradleGav
	Type string `yaml:"type" json:"type"`
	Excludes []string `yaml:"excludes" json:"excludes"`
}

type GradleRepository struct {
	Type string `yaml:"type" json:"type"`
	Data GradleRepositoryData `yaml:"data" json:"data"`
}

type GradleRepositoryData struct {
	Name string `yaml:"name" json:"name"`
}

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

func parseGardleBuildFile(buildFilePath string) (out map[string]interface{}, err error) {
	state := gradleParseState{}
	buildFile, err := os.ReadFile(buildFilePath)
	if err != nil {
		logrus.Errorf("Unable to read gradle build file : %s", err)
		return nil, err
	}
	return deepParse(buildFile, &state, false, true), nil
}

// Based on https://github.com/ninetwozero/gradle-to-js/blob/master/lib/parser.js
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
	charBlockend         = '}'

	keywordDef = "def"
	keywordIf  = "if"

	singleLineCommentStart = `//`
	blockCommentStart      = `/*`
	blockCommentEnd        = `*/`
)

var (
	depsKeywordString        = `(?m)\s*([A-Za-z0-9_-]+)\s*`
	depsKeywordStringPattern = regexp.MustCompile(depsKeywordString)
	depsEasyGavStringRegex   = regexp.MustCompile(`(["']?)([\w.-]+):([\w.-]+):([\w\[\]\(\),+.-]+)\1`)
	depsHardGavStringRegex   = regexp.MustCompile(depsKeywordString + `(?:\((.*)\)|(.*))`)
	depsItemBlockRegex       = regexp.MustCompile(depsKeywordString + `(?m)\(((["']?)(.*)\3)\)\s*\{`)
	depsExcludeLineRegex     = regexp.MustCompile(`exclude[ \\t]+([^\\n]+)`)
	pluginsLinePattern       = regexp.MustCompile(`(?m)(id|version)(?:\s)(["']?)([A-Za-z0-9.]+)\2`)

	functionRegex = regexp.MustCompile(`\w+\(.*\);?$`)

	specialKeys = map[string]func(chunk string, state gradleParseState){
		"repositories": parseRepositoryClosure,
		"dependencies": parseDependencyClosure,
		"plugins":      parsePluginsClosure,
	}

	whitespacecharacters = map[rune]bool{
		charTab:            true,
		charNewLine:        true,
		charCarriageReturn: true,
		charSpace:          true,
	}
)

//TODO: skipEmptyValues - default should be true
func deepParse(chunk string, state *gradleParseState, keepFunctionCalls, skipEmptyValues bool) (out map[string]interface{}) {
	var character rune
	var tempString, commentText string
	var currentKey string
	parsingKey := true
	isBeginningOfLine := true

	for chunkLength := len(chunk); state.index < chunkLength; state.index++ {
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
			if !currentKey && tempString {
				if parsingKey {
					if isFunctionCall(tempString) && !keepFunctionCalls {
						continue
					} else {
						currentKey = strings.TrimSpace(tempString)
						tempString = ""
					}
				}
			}
			if tempString || (currentKey && !skipEmptyValues) {
				addValueToStructure(out, currentKey, trimWrappingQuotes(tempString))
				currentKey = ""
				tempString = ""
			}
			parsingKey = true
			isBeginningOfLine = true

			state.comment.reset()
			continue
		}
		// Only parse as an array if the first *real* char is a [
		if !parsingKey && !tempString && character == charArrayStart {
			out[currentKey] = parseArray(chunk, state)
			currentKey = ""
			tempString = ""
			continue
		}
		if character == charBlockStart {
			// We need to skip the current (=start) character so that we literally "step into" the next closure/block
			state.index++
			if sk, ok := specialKeys[currentKey]; ok {
				out[currentKey] = sk(chunk, state)
			} else if out[currentKey] {
				out[currentKey] = deepAssign(map[string]string{}, out[currentKey], deepParse(chunk, state, keepFunctionCalls, skipEmptyValues))
			} else {
				out[currentKey] = deepParse(chunk, state, keepFunctionCalls, skipEmptyValues)
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
	addValueToStructure(out, currentKey, trimWrappingQuotes(tempString))
	return out
}

func parseRepositoryClosure(chunk string, state *gradleParseState) (repositories []GradleRepository) {
	repositories := []GradleRepository{}
	parsedRepos := deepParse(chunk, state, true, false)
	for parsedRepoType, value := range parsedRepos {
		if parsedRepos[item] == nil {
			reposiitories = append(repositories, GradleRepository{Type: parsedRepoType, Data: value})
		  } else {
			repositories = append(repositories, GradleRepository{Type: "unknown", Data: {Name: parsedRepoType}})
		  }
	}
	return repositories
}

type mappingFn func(string) string
  
  func parseSpecialClosure(chunk string, state *gradleParseState, mapFunction mappingFn) (out []string) {
	out := []string{}
	// openBlockCount starts at 1 due to us entering after "<key> {"
	var openBlockCount = 1
	var currentKey = ""
	var currentValue = ""
  
	var isInItemBlock = false;
	for ; state.index < chunk.length; state.index++ {
	  if chunk[state.index] == charBlockStart {
		openBlockCount++;
	  } else if chunk[state.index] == charBlockEnd {
		openBlockCount--;
	  } else {
		currentKey += String.fromCharCode(chunk[state.index]);
	  }
  
	  // Keys shouldn't have any leading nor trailing whitespace
	  currentKey = currentKey.trim();
  
	  if (isStartOfComment(currentKey)) {
		var commentText = currentKey;
		for state.index = state.index + 1; state.index < chunk.length; state.index++ {
		  if (isCommentComplete(commentText, chunk[state.index])) {
			currentKey = ""
			break;
		  }
		  commentText += String.fromCharCode(chunk[state.index]);
		}
	  }
  
  
	  if currentKey && isWhitespace(chunk[state.index]) {
		var character = ""
		for state.index = state.index + 1; state.index < chunk.length; state.index++ {
		  character = chunk[state.index];
		  currentValue += String.fromCharCode(character);
  
		  if character == charBlockStart {
			isInItemBlock = true;
		  } else if isInItemBlock && character == charBlockEnd {
			isInItemBlock = false;
		  } else if !isInItemBlock {
			if isLineBreakCharacter(character) && currentValue {
			  break;
			}
		  }
		}
  
		out = append(out, mapFunction(currentKey + ' ' + currentValue))
		currentKey = ""
		currentValue = ""
	  }
  
	  if (openBlockCount == 0) {
		break;
	  }
	}
	return out;
  }
  


func skipIfStatement(chunk string, state *gradleParseState) {
  skipFunctionCall(chunk, state)
  var character rune
  var hasFoundTheCurlyBraces, hasFoundAStatementWithoutBraces
  curlyBraceCount := 0
  for max := len(chunk); state.index < max; state.index++ {
    character = chunk[state.index]
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
  start := state.index
  parenthesisNest := 1
  state.index+=1
  for character := ([]rune(chunk))[state.index]; character != 0 && parenthesisNest; {
    if character == charLeftParanthesis {
      parenthesisNest++
    } else if character == charRightParanthesis {
      parenthesisNest--
    }
	state.index+=1
    character = chunk[state.index]
  }
  for ;character && character != charBlockStart; {
	  state.index+=1
    character = chunk[state.index]
  }
  state.index+=1
  character = chunk[state.index]
 blockNest := 1
  for ;character != 0 && blockNest; {
    if (character == charBlockStart) {
      blockNest++
    } else if character == charBlockEnd {
      blockNest--
    }
	state.index+=1
    character = chunk[state.index]
  }
  state.index-=1
}

func parseDependencyClosure(chunk, state) []string {
  return parseSpecialClosure(chunk, state, createStructureForDependencyItem)
}

func createStructureForDependencyItem(data string) GradleDependencyItem {
  gdi := GradleDependencyItem{}
  matches := depsItemBlockRegex.FindAll(data);
  if (matches && matches[2]) {
    excludes = []string{}
    var match
    while((match = depsExcludeLineRegex.exec(data))) {
      excludes.push(parseMapNotation(match[0].substring(findFirstSpaceOrTabPosition(match[0]))));
    }
    out = parseGavString(matches[2])
    out['type'] = matches[1]
    out['excludes'] = excludes
  } else {
    out = parseGavString(data)
    parsed := depsKeywordStringRegex.FindString(data)
    out['type'] = (parsed && parsed[1]) || ''
    out['excludes'] = []
  }
  return out
}

func parsePluginsClosure(chunk string, state *gradleParseState) []string {
  return parseSpecialClosure(chunk, state, createStructureForPlugin);
}

func createStructureForPlugin(pluginRow) {
  var out = {};

  var match;
  while(match = pluginsLinePattern. exec(pluginRow)) {
    if (match && match[1]) {
      out[match[1]] = match[3];
    }
  }
  return out;
}

func findFirstSpaceOrTabPosition(input) {
  var position = input.indexOf(' ');
  if (position === -1) {
    position = input.indexOf('\t');
  }
  return position;
}

func parseGavString(gavString) (out Gav) {
  out = Gav{}
  easyGavStringMatches := depsEasyGavStringRegex.FindAll(gavString)
  if len(easyGavStringMatches)!=0 {
    out.Group = easyGavStringMatches[2]
    out.Name = easyGavStringMatches[3]
    out.Version = easyGavStringMatches[4]
  } else if strings.Contains(gavString,`project(`) {
    out.Name = gavString.match(`(project\([^\)]+\))`)[0]
  } else {
    var hardGavMatches = depsHardGavStringRegex DEPS_HARD_GAV_STRING_REGEX.exec(gavString)
    if (hardGavMatches && (hardGavMatches[3] || hardGavMatches[2])) {
      out = parseMapNotationWithFallback(out, hardGavMatches[3] || hardGavMatches[2])
    } else {
      out = parseMapNotationWithFallback(out, gavString, gavString.slice(findFirstSpaceOrTabPosition(gavString)))
    }
  }
  return out
}

func parseMapNotationWithFallback(currMap map[string]string, str , name string) map[string]string {
  outFromMapNotation := parseMapNotation(str)
  if _, ok := outFromMapNotation["name"]; ok {
    return outFromMapNotation
  }
  if name == "" {
	  name = str
  }
  currMap["name"] = name
  return currMap
}

func parseMapNotation(input string) (outMap map[string]string) {
outMap := map[string]string
  currentKey := ""
  quotation := ""
  for i, max := 0,len(input); i < max; i++ {
    if input[i] == ':' {
      currentKey = strings.TrimSpace(currentKey)
      outMap[currentKey] = ''
      for innerLoop := rune(0), i = i + 1; i < max; i++ {
        if innerLoop == 0 {
          // Skip any leading spaces before the actual value
          if isWhitespaceLiteral(input[i]) {
            continue
          }
        }
        // We just take note of what the "latest" quote was so that we can
        if input[i] == '"' || input[i] == "'" {
          quotation = input[i]
          continue
        }
        // Moving on to the next value if we find a comma
        if input[i] === ',' {
			outMap[currentKey] = strings.TrimSpace(out[currentKey])
          currentKey = ''
          break
        }
        outMap[currentKey] += input[i]
        innerLoop++
      }
    } else {
      currentKey += input[i]
    }
  }
  // If the last character contains a quotation mark, we remove it
  if val, ok := outMap[currentKey]; ok {
    val = strings.TrimSuffix(strings.TrimSpace(val), quotation)
	outMap[currentKey] = val
  }
  return outMap
}

func fetchDefinedNameOrSkipFunctionDefinition(chunk string, state *gradleParseState) string {
  var character rune
  temp := ""
  isVariableDefinition := true
  for max := len([]rune(chunk)); state.index < max; state.index++ {
    character = chunk[state.index]

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
    values := strings.Split(strings.TrimSpace(temp),' ')
    return values[values.length - 1]
  } else {
    return ''
  }
}

func parseArray(chunk string, state *gradleParseState) []string {
  var character rune
  temp = ""
  for max := len([]rune(chunk)); state.index < max; state.index++ {
    character = chunk[state.index]
    if character == charArrayStart {
      continue
    } else if character == charArrayEnd {
      break
    }
    temp += string(character)
  }
  elems := strings.Split(temp, ",")
  for elemI, elem := range elems {
	  elems[elemI] = trimWrappingQuotes(strings.TrimSpace(elem)
  }
  return elems
}

func skipFunctionCall(chunk string, state *gradleParseState) bool {
	openParenthesisCount := 0
	var character rune
	for max := len(chunk); state.index < max; state.index++ {
		character = ([]rune(chunk))[state.index]
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

  func addValueToStructure(structure, currentKey, value) {
	if currentKey {
	  if structure.hasOwnProperty(currentKey) {
		if structure[currentKey].constructor == Array {
		  structure[currentKey].push(getRealValue(value));
		} else {
		  oldValue := structure[currentKey]
		  structure[currentKey] = [oldValue, getRealValue(value)];
		}
	  } else {
		structure[currentKey] = getRealValue(value);
	  }
	}
  }

  // FIXME
func getRealValue(value bool) bool {
	if value == true || value == false {
		return value == true
	}
	return value
}

func trimWrappingQuotes(str string) string {
	doubleQuote := `"`
	singleQuote := `'`
	if strings.HasPrefix(str, doubleQuote) {
		return strings.TrimPrefix(strings.TrimSuffix(str, doubleQuote), doubleQuote)
	} else if strings.HasPrefix(str, singleQuote) {
		return strings.TrimPrefix(strings.TrimSuffix(str, singleQuote), singleQuote)
	}
	return str
}

func isDelimiter(character rune) bool {
	return character == charSpace || character == charEquals
}

func isWhitespace(character rune) bool {
	return whitespacecharacters[character]
}

func isWhitespaceLiteral(character string) bool {
	return isWhitespace(rune(character[0]))
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
	return comment[len(comment)-2:] == blockCommentEnd
}

func isFunctionCall(str string) bool {
	return functionRegex.MatchString(str)
}
