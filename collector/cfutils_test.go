package collector_test

import (
	"testing"

	"github.com/cloudfoundry-community/go-cfclient/v2"
	"github.com/konveyor/move2kube/types"
	"github.com/stretchr/testify/assert"

	collector "github.com/konveyor/move2kube/collector"
)

func TestNewCfApps(t *testing.T) {
	cfapps := collector.NewCfApps()
	assert.Equal(t, string(collector.CfAppsMetadataKind), cfapps.Kind, "Failed to initialize CfApps kind properly.")
	assert.Equal(t, types.SchemeGroupVersion.String(), cfapps.APIVersion, "Failed to initialize CfApps APIVersion properly.")
}

func TestFormatMapsWithInterface(t *testing.T) {
	app := collector.CfApp{
		Application: collector.App{
			DockerCredentialsJSON: map[string]interface{}{"key1": "value1"},
			Environment:           map[string]interface{}{"env1": "value1"},
		},
		Environment: cfclient.AppEnv{
			Environment:    map[string]interface{}{"env2": "value2"},
			ApplicationEnv: map[string]interface{}{"appenv1": "value1"},
			RunningEnv:     map[string]interface{}{"runenv1": "value1"},
			StagingEnv:     map[string]interface{}{"stageenv1": "value1"},
			SystemEnv:      map[string]interface{}{"sysenv1": "value1"},
		},
	}
	cfApps := collector.CfApps{
		Spec: collector.CfAppsSpec{
			CfApps: []collector.CfApp{app},
		},
	}

	cfApps = collector.FormatMapsWithInterface(cfApps)
	expectedApp := collector.CfApp{
		Application: collector.App{
			DockerCredentialsJSON: map[string]interface{}{"key1": "value1"},
			Environment:           map[string]interface{}{"env1": "value1"},
		},
		Environment: cfclient.AppEnv{
			Environment:    map[string]interface{}{"env2": "value2"},
			ApplicationEnv: map[string]interface{}{"appenv1": "value1"},
			RunningEnv:     map[string]interface{}{"runenv1": "value1"},
			StagingEnv:     map[string]interface{}{"stageenv1": "value1"},
			SystemEnv:      map[string]interface{}{"sysenv1": "value1"},
		},
	}
	expectedCfApps := collector.CfApps{
		Spec: collector.CfAppsSpec{
			CfApps: []collector.CfApp{expectedApp},
		},
	}
	assert.Equal(t, expectedCfApps, cfApps, "Failed to format maps with interface correctly.")
}
