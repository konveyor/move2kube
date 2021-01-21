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
	"github.com/konveyor/move2kube/internal/move2kube"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type planFlags struct {
	planfile string
	srcpath  string
	name     string
}

func planHandler(flags planFlags) {
	// Check if this is even a directory
	var err error
	planfile := flags.planfile
	srcpath := flags.srcpath
	name := flags.name

	planfile, err = filepath.Abs(planfile)
	if err != nil {
		log.Fatalf("Failed to make the plan file path %q absolute. Error: %q", planfile, err)
	}
	srcpath, err = filepath.Abs(srcpath)
	if err != nil {
		log.Fatalf("Failed to make the source directory path %q absolute. Error: %q", srcpath, err)
	}
	// TODO: should we normalize the project name?
	fi, err := os.Stat(srcpath)
	if err != nil {
		log.Fatalf("Unable to access source directory : %s", err)
	}
	if !fi.IsDir() {
		log.Fatalf("Input is a file, expected directory: %s", srcpath)
	}
	fi, err = os.Stat(planfile)
	if os.IsNotExist(err) {
		if strings.HasSuffix(planfile, string(os.PathSeparator)) {
			planfile = filepath.Join(planfile, common.DefaultPlanFile)
		} else if !strings.Contains(filepath.Base(planfile), ".") {
			planfile = filepath.Join(planfile, common.DefaultPlanFile)
		}
	} else if err != nil {
		log.Fatalf("Error while accessing plan file path %s : %s ", planfile, err)
	} else if fi.IsDir() {
		planfile = filepath.Join(planfile, common.DefaultPlanFile)
	}

	p := move2kube.CreatePlan(srcpath, name, false)
	if err = plantypes.WritePlan(planfile, p); err != nil {
		log.Errorf("Unable to write plan file (%s) : %s", planfile, err)
		return
	}
	log.Infof("Plan can be found at [%s].", planfile)
}

func getPlanCommand() *cobra.Command {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	viper.AutomaticEnv()

	flags := planFlags{}
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Plan out a move",
		Long:  "Discover and create a plan file based on an input directory",
		Run:   func(*cobra.Command, []string) { planHandler(flags) },
	}

	planCmd.Flags().StringVarP(&flags.srcpath, cmdcommon.SourceFlag, "s", ".", "Specify source directory.")
	planCmd.Flags().StringVarP(&flags.planfile, cmdcommon.PlanFlag, "p", common.DefaultPlanFile, "Specify a file path to save plan to.")
	planCmd.Flags().StringVarP(&flags.name, cmdcommon.NameFlag, "n", common.DefaultProjectName, "Specify the project name.")

	must(planCmd.MarkFlagRequired(cmdcommon.SourceFlag))

	return planCmd
}
