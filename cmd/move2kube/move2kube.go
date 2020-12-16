/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"

	"github.com/konveyor/move2kube/internal/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var verbose bool

// RootCmd root level flags and commands
var rootCmd = &cobra.Command{
	Use:   "move2kube",
	Short: "A tool to modernize to kubernetes/openshift",
	Long:  `move2kube is a tool to help optimally translate from platforms such as docker-swarm, CF to Kubernetes.`,
	PersistentPreRunE: func(*cobra.Command, []string) error {
		if verbose {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

func main() {
	assetsPath, tempPath, err := common.CreateAssetsData()
	if err != nil {
		log.Fatalf("Unable to create the assets directory. Error: %q", err)
	}
	common.TempPath = tempPath
	common.AssetsPath = assetsPath
	defer os.RemoveAll(tempPath)
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %q", err)
	}
}
