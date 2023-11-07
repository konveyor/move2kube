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

package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube-wasm/lib"
	"github.com/konveyor/move2kube-wasm/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type collectFlags struct {
	annotations string
	outpath     string
	srcpath     string
}

func collectHandler(flags collectFlags) {
	var err error
	annotations := flags.annotations
	outpath := flags.outpath
	srcpath := flags.srcpath

	if outpath != "" {
		if outpath, err = filepath.Abs(outpath); err != nil {
			logrus.Fatalf("Failed to make the output directory path '%s' absolute. Error: %q", outpath, err)
		}
	}
	if srcpath != "" {
		srcpath, err = filepath.Abs(srcpath)
		if err != nil {
			logrus.Fatalf("Failed to make the source directory path '%s' absolute. Error: %q", srcpath, err)
		}
		fi, err := os.Stat(srcpath)
		if os.IsNotExist(err) {
			logrus.Fatalf("Source directory '%s' does not exist. Error: %q", srcpath, err)
		} else if err != nil {
			logrus.Fatalf("Error while accessing directory: '%s' . Error: %q", srcpath, err)
		} else if !fi.IsDir() {
			logrus.Fatalf("Source path is a file, expected '%s' to be a directory.", srcpath)
		}
	}
	outpath = filepath.Join(filepath.Clean(outpath), types.AppNameShort+"_collect")
	if annotations == "" {
		lib.Collect(srcpath, outpath, []string{})
	} else {
		lib.Collect(srcpath, outpath, strings.Split(annotations, ","))
	}
	logrus.Infof("Collect Output in [%s]. Copy this directory into the source directory to be used for planning.", outpath)
}

// GetCollectCommand returns a command to collect information from running applications
func GetCollectCommand() *cobra.Command {
	viper.AutomaticEnv()

	flags := collectFlags{}
	collectCmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect and process metadata from multiple sources.",
		Long:  "Collect metadata from multiple sources (cluster, image repo etc.), filter and summarize it into a yaml.",
		Run:   func(*cobra.Command, []string) { collectHandler(flags) },
	}

	collectCmd.Flags().StringVarP(&flags.annotations, "annotations", "a", "", "Specify annotations to select collector subset.")
	collectCmd.Flags().StringVarP(&flags.outpath, outputFlag, "o", ".", "Specify output directory for collect.")
	collectCmd.Flags().StringVarP(&flags.srcpath, sourceFlag, "s", "", "Specify source directory for the artifacts to be considered while collecting.")

	return collectCmd
}
