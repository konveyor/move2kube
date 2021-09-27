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
	"io"
	"os"

	"github.com/konveyor/move2kube/assets"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func main() {
	loglevel := logrus.InfoLevel.String()
	logFile := ""

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
			logl, err := logrus.ParseLevel(loglevel)
			if err != nil {
				logrus.Errorf("the log level '%s' is invalid, using 'info' log level instead. Error: %q", loglevel, err)
				logl = logrus.InfoLevel
			}
			logrus.SetLevel(logl)
			if logFile != "" {
				f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE, common.DefaultFilePermission)
				if err != nil {
					logrus.Fatalf("failed to open the log file at path %s . Error: %q", logFile, err)
				}
				logrus.SetOutput(io.MultiWriter(f, os.Stdout))
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&loglevel, "log-level", logrus.InfoLevel.String(), "Set logging levels.")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "File to store the logs in. By default it only prints to console.")

	rootCmd.AddCommand(getVersionCommand())
	rootCmd.AddCommand(getCollectCommand())
	rootCmd.AddCommand(getPlanCommand())
	rootCmd.AddCommand(getTransformCommand())
	rootCmd.AddCommand(getParameterizeCommand())

	assetsFilePermissions := map[string]int{}
	err := yaml.Unmarshal([]byte(assets.AssetFilePermissions), &assetsFilePermissions)
	if err != nil {
		logrus.Fatalf("unable to convert permissions. Error: %q", err)
	}
	assetsPath, tempPath, err := common.CreateAssetsData(assets.AssetsDir, assetsFilePermissions)
	if err != nil {
		logrus.Fatalf("unable to create the assets directory. Error: %q", err)
	}
	common.TempPath = tempPath
	common.AssetsPath = assetsPath
	defer os.RemoveAll(tempPath)
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("Error: %q", err)
	}
}
