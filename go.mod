module github.com/konveyor/move2kube

go 1.15

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20200131002437-cf55d5288a48 // indirect
	code.cloudfoundry.org/cli v7.1.0+incompatible
	github.com/AlecAivazis/survey/v2 v2.2.3
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/charlievieth/fs v0.0.1 // indirect
	github.com/cloudfoundry/bosh-cli v6.4.1+incompatible
	github.com/cloudfoundry/bosh-utils v0.0.0-20201128100250-4ba3e9fa0520 // indirect
	github.com/containers/skopeo v1.2.0
	github.com/cppforlife/go-patch v0.2.0 // indirect
	github.com/docker/cli v20.10.0-rc1+incompatible
	github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/docker/libcompose v0.4.1-0.20171025083809-57bd716502dc
	github.com/go-git/go-git/v5 v5.2.0
	github.com/google/go-cmp v0.5.4
	github.com/gorilla/mux v1.8.0
	github.com/moby/buildkit v0.7.2
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20200728170252-4d89ac9fbff6
	github.com/openshift/api v0.0.0-20200930075302-db52bc4ef99f // release-4.6
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/tektoncd/pipeline v0.18.1
	github.com/tektoncd/triggers v0.10.0
	github.com/whilp/git-urls v1.0.0
	github.com/xrash/smetrics v0.0.0-20200730060457-89a2a8a1fb0b
	golang.org/x/crypto v0.0.0-20201124201722-c8d3bf9c5392
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/kubernetes v1.19.4
	knative.dev/serving v0.19.0
)

replace (
	github.com/containerd/containerd v1.4.0-0 => github.com/containerd/containerd v1.4.0
	github.com/docker/cli => github.com/docker/cli v0.0.0-20200210162036-a4bedce16568
	github.com/docker/docker v0.0.0 => github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/xeipuuv/gojsonschema => github.com/xeipuuv/gojsonschema v0.0.0-20161030231247-84d19640f6a7 // indirect
	k8s.io/api => k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.4
	k8s.io/apiserver => k8s.io/apiserver v0.19.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.4
	k8s.io/client-go => k8s.io/client-go v0.19.4
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.4
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.4
	k8s.io/code-generator => k8s.io/code-generator v0.19.4
	k8s.io/component-base => k8s.io/component-base v0.19.4
	k8s.io/cri-api => k8s.io/cri-api v0.19.4
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.4
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.4
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.4
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.4
	k8s.io/kubectl => k8s.io/kubectl v0.19.4
	k8s.io/kubelet => k8s.io/kubelet v0.19.4
	k8s.io/kubernetes => k8s.io/kubernetes v1.19.4
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.4
	k8s.io/metrics => k8s.io/metrics v0.19.4
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.4
)
