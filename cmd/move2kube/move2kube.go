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
	"path/filepath"
	"strings"

	cmdcommon "github.com/konveyor/move2kube/cmd/common"
	"github.com/konveyor/move2kube/internal/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	verbose := false

	// RootCmd root level flags and commands
	rootCmd := &cobra.Command{
		Use:   "move2kube",
		Short: "Move2Kube creates all the resources required for deploying your application into kubernetes, including containerisation and kubernetes resources.",
		Long: `Move2Kube creates all resources required for deploying your application into kubernetes, including containerisation and kubernetes resources.
It supports translating from docker swarm/docker-compose, cloud foundry apps and even other non-containerized applications.
Even if the app does not use any of the above, or even if it is not containerized it can still be translated.

For more documentation and support, visit https://move2kube.konveyor.io/
`,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			if verbose {
				log.SetLevel(log.DebugLevel)
			}
			return nil
		},
	}

	if strings.HasPrefix(filepath.Base(os.Args[0]), "kubectl-") {
		// Invoked as a kubectl plugin.

		// Cobra doesn't have a way to specify a two word command (ie. "kubectl move2kube"), so set a custom usage template
		// with kubectl in it. Cobra will use this template for the root and all child commands.
		oldUsageTemplate := rootCmd.UsageTemplate()
		newUsageTemplate := strings.NewReplacer("{{.UseLine}}", "kubectl {{.UseLine}}", "{{.CommandPath}}", "kubectl {{.CommandPath}}").Replace(oldUsageTemplate)
		rootCmd.SetUsageTemplate(newUsageTemplate)
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.AddCommand(cmdcommon.GetVersionCommand())
	rootCmd.AddCommand(getCollectCommand())
	rootCmd.AddCommand(getPlanCommand())
	rootCmd.AddCommand(getTranslateCommand())
	rootCmd.AddCommand(getValidateCommand())
	rootCmd.AddCommand(getParameterizeCommand())

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
