/*
 *  Copyright IBM Corporation 2020, 2021
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
	"os"

	"github.com/konveyor/move2kube/assets"
	"github.com/konveyor/move2kube/cmd"
	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func main() {
	rootCmd := cmd.GetRootCmd()
	assetsFilePermissions := map[string]int{}
	err := yaml.Unmarshal([]byte(assets.AssetFilePermissions), &assetsFilePermissions)
	if err != nil {
		logrus.Fatalf("failed to unmarshal the assets permissions file as YAML. Error: %q", err)
	}
	assetsPath, tempPath, remoteTempPath, err := common.CreateAssetsData(assets.AssetsDir, assetsFilePermissions)
	if err != nil {
		logrus.Fatalf("failed to create the assets directory. Error: %q", err)
	}
	common.TempPath = tempPath
	common.AssetsPath = assetsPath
	common.RemoteTempPath = remoteTempPath
	defer os.RemoveAll(tempPath)
	defer os.RemoveAll(remoteTempPath)
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("Error: %q", err)
	}
}
