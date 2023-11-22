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

package qaengine

import (
	"fmt"

	tsize "github.com/kopoli/go-terminal-size"
)

// AddRightAlignedString adds a new string to the right of the original, with max width the size of the current
// terminal window
func AddRightAlignedString(original, addition string) string {
	termSize, err := tsize.GetSize()
	if err != nil {
		termSize = tsize.Size{
			Height: 100, // the height here doesn't matter
			Width:  100, // TODO: is 100 a good default for terminal width?
		}
	}
	width := termSize.Width - len(original)
	return fmt.Sprintf("%s%*s", original, width, addition)
}
