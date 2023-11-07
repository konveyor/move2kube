package lib

import (
	"github.com/konveyor/move2kube-wasm/types/info"
	"gopkg.in/yaml.v3"
)

// GetVersion returns the version
func GetVersion(long bool) string {
	if !long {
		return info.GetVersion()
	}
	v := info.GetVersionInfo()
	ver, _ := yaml.Marshal(v)
	return string(ver)
}
