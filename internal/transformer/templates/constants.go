// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots at
// 2021-01-19 14:57:48.563204 +0900 JST m=+0.007877624

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

package templates

const (
	Buildimages_sh = `#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

{{range $key, $val := .}}
cd {{$val}}
./{{$key}}
cd -{{end}}
`

	Chart_tpl = `name: {{.Name}}
description: A generated Helm Chart for {{.Name}} 
version: 0.1.0
apiVersion: v1
keywords:
  - {{.Name}}
sources:
home:`

	CopySources_sh = `#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

print_usage() {
    echo "Invalid args: $*"
    echo 'Usage: ./scripts/copysources.sh path/to/sources/'
    echo 'Example: ./scripts/copysources.sh {{ .RelRootDir }}'
}

if [ "$#" -ne 1 ]; then
    print_usage "$@"
    exit 1
fi

cp -r "$1"/* {{.Dst}}/
`

	DeployCICD_sh = `#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

kubectl apply -f cicd/
{{ if .IsBuildConfig }}
HOST_AND_PORT="$(kubectl config view --minify -o=jsonpath='{.clusters[0].cluster.server}')"
NAMESPACE="$(kubectl config view --minify -o=jsonpath='{.contexts[0].context.namespace}')"
echo 'Please add the following web hooks to the corresponding git repositories:'
{{range $gitRepoURL, $webHookURL := .GitRepoToWebHookURLs}}
{{range $webHookURL}}echo "{{ $gitRepoURL }} : {{ . }}"
{{end}}
{{end}}
{{end}}
`

	Deploy_sh = `#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

kubectl apply -f {{ .Project }}
cat NOTES.txt
`

	HelmNotes_txt = `
Thank you for installing {{ .Chart.Name }}. Your release is named {{ .Release.Name }}.
{{ if .Values.ingresshost }}
Your services are exposed in ingress at {{ .Release.Name }}-{{ .Values.ingresshost }}.
{{ end }}
To learn more about the release, try:
  $ helm status {{ .Release.Name }}
  $ helm get all {{ .Release.Name }}`

	Helminstall_sh = `#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

helm upgrade -i {{ .Project }} {{ .Project }}`

	K8sReadme_md = `Move2Kube
---------
Congratulations! Move2Kube has generated the necessary build artfiacts for moving all your application components to Kubernetes. Using the artifacts in this directory you can deploy your application in a kubernetes cluster.

Prerequisites
-------------
* Docker
* Helm
* Kubectl
* Source-To-Image (S2I) https://github.com/openshift/source-to-image

Next Steps
----------
{{- if .NewImages }}
{{- if .AddCopySources}}
* Copy the source directory into the containers folder for packaging as containers using "./scripts/copysource.sh <SRC_DIR>"
{{- end }}
* Build your images using "./scripts/buildimages.sh"
* Push images to registry "./scripts/pushimages.sh <REGISTRY_URL> <REGISTRY_NAMESPACE>"
{{- end}}
{{- if .Helm }}
* Your helm chart is at {{ .Project }}, you can install it using "./scripts/helminstall.sh" or you can use the operator.
{{- else }}
* Use "deploy.sh" to deploy your artifacts into a kubernetes cluster.
{{- end }}
`

	KnativeReadme_md = `Move2Kube
---------
Congratulations! Move2Kube has generated the necessary build artifacts for moving all your application components to Knative. Using the artifacts in this directory you can deploy your application in a Knative instance

Prerequisites
-------------
* Docker
* Kubectl

Next Steps
----------
{{- if .NewImages }}
{{- if .AddCopySources }}
* Copy the source directory into the containers folder for packaging as containers using "./scripts/copysource.sh <SRC_DIR>"
{{- end }}
* Build your images using ./scripts/buildimages.sh
* Push images to registry ./scripts/pushimages.sh
{{- end }}
* Use ./scripts/deploy.sh to deploy your artifacts into a knative.
`

	Manualimages_md = `Manual containers
-----------------
There is no known automated containerization approach for the below container requirements.

{{range $image := .Images}}{{$image}}
{{end}}`

	NOTES_txt = `{{if .IsHelm}}
{{if .ExposedServicePaths}}
The services are accessible on the following paths:
{{range $serviceName, $servicePath := .ExposedServicePaths}}{{ $serviceName }} : http://{{"{{ .Release.Name }}-{{ .Values.ingresshost }}"}}{{ $servicePath }}
{{end}}
{{else}}
This app has no exposed services.
{{end}}
{{else}}
{{ $baseURL := .IngressHost }}
{{if .ExposedServicePaths}}
The services are accessible on the following paths:
{{range $serviceName, $servicePath := .ExposedServicePaths}}{{ $serviceName }} : http://{{ $baseURL }}{{ $servicePath }}
{{end}}
{{else}}
This app has no exposed services.
{{end}}
{{end}}
`

	Pushimages_sh = `#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

# Invoke as pushimages.sh <registry_url> <registry_namespace>

if [ "$#" -ne 2 ]; then
    REGISTRY_URL={{ .RegistryURL }}
    REGISTRY_NAMESPACE={{ .RegistryNamespace }}
else
    REGISTRY_URL=$1
    REGISTRY_NAMESPACE=$2
fi

# Uncomment the below line if you want to enable login before pushing
# docker login ${REGISTRY_URL}

{{range $image := .Images}}docker tag {{$image}} ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/{{$image}}
docker push ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/{{$image}}
{{end}}`
)
