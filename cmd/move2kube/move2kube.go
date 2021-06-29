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
	cmdcommon "github.com/konveyor/move2kube/cmd/common"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func main() {
	verbose := false

	// RootCmd root level flags and commands
	rootCmd := &cobra.Command{
		Use:   "move2kube",
		Short: "Move2Kube creates all the resources required for deploying your application into kubernetes, including containerisation and kubernetes resources.",
		Long: `Move2Kube creates all resources required for deploying your application into kubernetes, including containerisation and kubernetes resources.
It supports translating from docker swarm/docker-compose, cloud foundry apps and even other non-containerized applications.
Even if the app does not use any of the above, or even if it is not containerized it can still be transformed.

For more documentation and support, visit https://move2kube.konveyor.io/
`,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if verbose {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.AddCommand(cmdcommon.GetVersionCommand())
	rootCmd.AddCommand(getCollectCommand())
	rootCmd.AddCommand(getPlanCommand())
	rootCmd.AddCommand(getTransformCommand())
	rootCmd.AddCommand(getValidateCommand())

	assetsFilePermissions := map[string]int{}
	err := yaml.Unmarshal([]byte(assets.AssetFilePermissions), &assetsFilePermissions)
	if err != nil {
		logrus.Fatalf("Unable to convert permissions : %s", err)
	}
	assetsPath, tempPath, err := common.CreateAssetsData(assets.AssetsDir, assetsFilePermissions)
	if err != nil {
		logrus.Fatalf("Unable to create the assets directory. Error: %q", err)
	}
	common.TempPath = tempPath
	common.AssetsPath = assetsPath
	defer os.RemoveAll(tempPath)
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("Error: %q", err)
	}
}
