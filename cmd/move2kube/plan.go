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

package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/lib"
	"github.com/konveyor/move2kube/qaengine"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type planFlags struct {
	progressServerPort int
	planfile           string
	srcpath            string
	name               string
	customizationsPath string
	//Configs contains a list of config files
	configs []string
	//Configs contains a list of key-value configs
	setconfigs []string
	//PreSets contains a list of preset configurations
	preSets []string
}

func planHandler(cmd *cobra.Command, flags planFlags) {
	ctx, cancel := context.WithCancel(cmd.Context())
	logrus.AddHook(common.NewCleanupHook(cancel))
	logrus.AddHook(common.NewCleanupHook(lib.Destroy))
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	go func() {
		<-ctx.Done()
		lib.Destroy()
		stop()
		common.Interrupt()
	}()
	defer lib.Destroy()

	var err error
	planfile := flags.planfile
	srcpath := flags.srcpath
	name := flags.name
	customizationsPath := flags.customizationsPath

	planfile, err = filepath.Abs(planfile)
	if err != nil {
		logrus.Fatalf("Failed to make the plan file path %q absolute. Error: %q", planfile, err)
	}
	srcpath, err = filepath.Abs(srcpath)
	if err != nil {
		logrus.Fatalf("Failed to make the source directory path %q absolute. Error: %q", srcpath, err)
	}
	fi, err := os.Stat(srcpath)
	if err != nil {
		logrus.Fatalf("Unable to access source directory : %s", err)
	}
	if !fi.IsDir() {
		logrus.Fatalf("Input is a file, expected directory: %s", srcpath)
	}
	fi, err = os.Stat(planfile)
	if os.IsNotExist(err) {
		if strings.HasSuffix(planfile, string(os.PathSeparator)) {
			planfile = filepath.Join(planfile, common.DefaultPlanFile)
		} else if !strings.Contains(filepath.Base(planfile), ".") {
			planfile = filepath.Join(planfile, common.DefaultPlanFile)
		}
	} else if err != nil {
		logrus.Fatalf("Error while accessing plan file path %s : %s ", planfile, err)
	} else if fi.IsDir() {
		planfile = filepath.Join(planfile, common.DefaultPlanFile)
	}
	qaengine.StartEngine(true, 0, true)
	qaengine.SetupConfigFile("", flags.setconfigs, flags.configs, flags.preSets)
	if flags.progressServerPort != 0 {
		startPlanProgressServer(flags.progressServerPort)
	}
	p := lib.CreatePlan(ctx, srcpath, "", customizationsPath, name)
	if err = plantypes.WritePlan(planfile, p); err != nil {
		logrus.Errorf("Unable to write plan file (%s) : %s", planfile, err)
		return
	}
	logrus.Infof("Plan can be found at [%s].", planfile)
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
		Run:   func(cmd *cobra.Command, _ []string) { planHandler(cmd, flags) },
	}

	planCmd.Flags().StringVarP(&flags.srcpath, sourceFlag, "s", ".", "Specify source directory.")
	planCmd.Flags().StringVarP(&flags.planfile, planFlag, "p", common.DefaultPlanFile, "Specify a file path to save plan to.")
	planCmd.Flags().StringVarP(&flags.name, nameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	planCmd.Flags().StringVarP(&flags.customizationsPath, customizationsFlag, "c", "", "Specify directory where customizations are stored.")
	planCmd.Flags().StringSliceVarP(&flags.configs, configFlag, "f", []string{}, "Specify config file locations")
	planCmd.Flags().StringSliceVar(&flags.preSets, preSetFlag, []string{}, "Specify preset config to use")
	planCmd.Flags().StringArrayVar(&flags.setconfigs, setConfigFlag, []string{}, "Specify config key-value pairs")

	planCmd.Flags().IntVar(&flags.progressServerPort, planProgressPortFlag, 0, "Port for the plan progress server. If not provided, the server won't be started.")

	must(planCmd.MarkFlagRequired(sourceFlag))
	must(planCmd.Flags().MarkHidden(planProgressPortFlag))

	return planCmd
}
