module github.com/konveyor/move2kube

go 1.16

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20210608160410-67692ebc98de // indirect
	code.cloudfoundry.org/cli v7.1.0+incompatible
	github.com/AlecAivazis/survey/v2 v2.2.12
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/apex/log v1.3.0 // indirect
	github.com/cloudfoundry/bosh-cli v6.4.1+incompatible
	github.com/cloudfoundry/bosh-utils v0.0.263 // indirect
	github.com/cloudrecipes/packagejson v1.0.0
	github.com/containerd/containerd v1.5.1 // indirect
	github.com/cppforlife/go-patch v0.2.0 // indirect
	github.com/dchest/uniuri v0.0.0-20200228104902-7aecb25e1fe5
	github.com/docker/cli v20.10.7+incompatible
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/libcompose v0.4.1-0.20171025083809-57bd716502dc
	github.com/go-git/go-git/v5 v5.4.2
	github.com/google/go-cmp v0.5.6
	github.com/gorilla/mux v1.8.0
	github.com/mattn/go-shellwords v1.0.12 // indirect
	github.com/mikefarah/yq/v4 v4.9.6
	github.com/mitchellh/mapstructure v1.4.1
	github.com/moby/buildkit v0.8.3
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/openshift/api v0.0.0-20210621070332-ea13be50ba2d
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/pkg/errors v0.9.1
	github.com/qri-io/starlib v0.5.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cast v1.3.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.8.0
	github.com/tektoncd/pipeline v0.25.0
	github.com/tektoncd/triggers v0.14.2
	github.com/whilp/git-urls v1.0.0
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673
	go.starlark.net v0.0.0-20210602144842-1cdb82c9e17a
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/mod v0.4.2 // indirect
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/op/go-logging.v1 v1.0.0-20160211212156-b2cb9fa56473
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/kubernetes v1.21.2
	knative.dev/serving v0.23.1
)

replace (
	github.com/containerd/containerd v1.4.0-0 => github.com/containerd/containerd v1.4.0
	github.com/docker/cli => github.com/docker/cli v0.0.0-20200210162036-a4bedce16568
	github.com/docker/docker v0.0.0 => github.com/docker/docker v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	github.com/xeipuuv/gojsonschema => github.com/xeipuuv/gojsonschema v0.0.0-20161030231247-84d19640f6a7 // indirect
	k8s.io/api => k8s.io/api v0.21.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.2
	k8s.io/apiserver => k8s.io/apiserver v0.21.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.2
	k8s.io/client-go => k8s.io/client-go v0.21.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.21.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.21.2
	k8s.io/code-generator => k8s.io/code-generator v0.21.2
	k8s.io/component-base => k8s.io/component-base v0.21.2
	k8s.io/component-helpers => k8s.io/component-helpers v0.21.2
	k8s.io/controller-manager => k8s.io/controller-manager v0.21.2
	k8s.io/cri-api => k8s.io/cri-api v0.21.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.21.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.21.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.21.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.21.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.21.2
	k8s.io/kubectl => k8s.io/kubectl v0.21.2
	k8s.io/kubelet => k8s.io/kubelet v0.21.2
	k8s.io/kubernetes => k8s.io/kubernetes v1.21.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.21.2
	k8s.io/metrics => k8s.io/metrics v0.21.2
	k8s.io/mount-utils => k8s.io/mount-utils v0.21.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.4
)
