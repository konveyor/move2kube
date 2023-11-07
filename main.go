package main

import (
	"github.com/konveyor/move2kube-wasm/assets"
	"github.com/konveyor/move2kube-wasm/cmd"
	"github.com/konveyor/move2kube-wasm/common"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	logrus.Infof("start")
	planCmd := cmd.GetPlanCommand()
	transformCmd := cmd.GetTransformCommand()
	transformCmd.SetArgs([]string{
		"--qa-skip",
	})
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
	if err := planCmd.Execute(); err != nil {
		logrus.Fatalf("Error: %q", err)
	}
	if err := transformCmd.Execute(); err != nil {
		logrus.Fatalf("Error: %q", err)
	}
	logrus.Infof("end")
}
