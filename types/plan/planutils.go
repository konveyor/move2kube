package plan

import (
	"github.com/konveyor/move2kube-wasm/common/deepcopy"
	"github.com/konveyor/move2kube-wasm/common/pathconverters"
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube-wasm/common"
	// "github.com/konveyor/move2kube/common/deepcopy"
	// "github.com/konveyor/move2kube/common/pathconverters"
	// "github.com/konveyor/move2kube/common/vcs"
	"github.com/sirupsen/logrus"
)

// ReadPlan decodes the plan from yaml converting relative paths to absolute.
func ReadPlan(path string, sourceDir string) (Plan, error) {
	plan := Plan{}
	var err error
	absSourceDir := ""
	if err = common.ReadMove2KubeYaml(path, &plan); err != nil {
		logrus.Errorf("Failed to load the plan file at path %q Error %q", path, err)
		return plan, err
	}
	if sourceDir != "" {
		plan.Spec.SourceDir = sourceDir
	}
	if plan.Spec.SourceDir != "" {
		//TODO: WASI
		//remoteSrcPath := vcs.GetClonedPath(plan.Spec.SourceDir, common.RemoteSourcesFolder, false)
		//if remoteSrcPath != "" {
		//	plan.Spec.SourceDir = remoteSrcPath
		//}
		absSourceDir, err = filepath.Abs(plan.Spec.SourceDir)
		if err != nil {
			logrus.Errorf("Unable to convert sourceDir to full path : %s", err)
			return plan, err
		}
	}
	if err = pathconverters.MakePlanPathsAbsolute(&plan, absSourceDir, common.TempPath); err != nil {
		return plan, err
	}
	plan.Spec.SourceDir = absSourceDir
	return plan, err
}

// WritePlan encodes the plan to yaml converting absolute paths to relative.
func WritePlan(path string, plan Plan) error {
	inputFSPath := plan.Spec.SourceDir
	//TODO: WASI
	// remoteSrcPath := vcs.GetClonedPath(plan.Spec.SourceDir, common.RemoteSourcesFolder, false)
	// if remoteSrcPath != "" {
	// 	inputFSPath = remoteSrcPath
	// }
	newPlan := deepcopy.DeepCopy(plan).(Plan)
	if err := pathconverters.ChangePaths(&newPlan, map[string]string{inputFSPath: "", common.TempPath: ""}); err != nil {
		logrus.Errorf("Unable to convert plan to use relative paths : %s", err)
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Unable to get current working dir : %s", err)
		return err
	}
	// if remoteSrcPath == "" && plan.Spec.SourceDir != "" {
	if plan.Spec.SourceDir != "" {
		if newPlan.Spec.SourceDir, err = filepath.Rel(wd, plan.Spec.SourceDir); err != nil {
			return err
		}
	}
	return common.WriteYaml(path, newPlan)
}
