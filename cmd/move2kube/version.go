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
	"fmt"

	"github.com/konveyor/move2kube/internal/move2kube"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	viper.AutomaticEnv()

	long := false
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the client version information",
		Long:  "Print the client version information",
		Run:   func(*cobra.Command, []string) { fmt.Println(move2kube.GetVersion(long)) },
	}

	versionCmd.Flags().BoolVarP(&long, "long", "l", false, "print the version details")

	rootCmd.AddCommand(versionCmd)
}
