module github.com/konveyor/move2kube

go 1.15

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20200131002437-cf55d5288a48 // indirect
	code.cloudfoundry.org/cli v7.1.0+incompatible
	github.com/AlecAivazis/survey/v2 v2.1.1
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/bmatcuk/doublestar v1.3.2 // indirect
	github.com/charlievieth/fs v0.0.0-20170613215519-7dc373669fa1 // indirect
	github.com/cloudfoundry/bosh-cli v6.4.0+incompatible
	github.com/cloudfoundry/bosh-utils v0.0.0-20200909190919-f6fb70428c10 // indirect
	github.com/containers/skopeo v1.1.1-0.20200811214205-ea10e61f7d60
	github.com/cppforlife/go-patch v0.2.0 // indirect
	github.com/docker/cli v0.0.0-20200227165822-2298e6a3fe24
	github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible // indirect
	github.com/docker/libcompose v0.4.1-0.20171025083809-57bd716502dc
	github.com/go-git/go-git/v5 v5.1.0
	github.com/gorilla/mux v1.8.0
	github.com/moby/buildkit v0.7.2
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20200728170252-4d89ac9fbff6
	github.com/openshift/api v0.0.0-20200326160804-ecb9283fe820 // 4.1
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.1
	github.com/tektoncd/pipeline v0.16.3
	github.com/xrash/smetrics v0.0.0-20200730060457-89a2a8a1fb0b
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/serving v0.17.2
)

replace (
	github.com/containerd/containerd v1.4.0-0 => github.com/containerd/containerd v1.4.0
	github.com/docker/cli => github.com/docker/cli v0.0.0-20200210162036-a4bedce16568
	github.com/docker/docker v0.0.0 => github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/xeipuuv/gojsonschema => github.com/xeipuuv/gojsonschema v0.0.0-20161030231247-84d19640f6a7 // indirect
	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
)
