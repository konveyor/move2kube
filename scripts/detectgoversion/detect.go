// +build ignore

/*
 *  Copyright IBM Corporation 2020
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

import (
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/mod/modfile"
)

func main() {
	modFilePath := "go.mod"
	if len(os.Args) == 2 && os.Args[1] != "" {
		modFilePath = os.Args[1]
	}
	data, err := ioutil.ReadFile(modFilePath)
	if err != nil {
		panic(err)
	}
	modFile, err := modfile.Parse(modFilePath, data, nil)
	if err != nil {
		panic(err)
	}
	if modFile.Go == nil {
		panic(fmt.Sprintf("didn't find the go version in the go.mod file at path %s", modFilePath))
	}
	fmt.Print(modFile.Go.Version)
}
