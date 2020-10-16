// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots at
// 2020-10-16 15:00:12.935031 +0530 IST m=+0.005946390

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
cp -r {{.Src}}/* {{.Dst}}/
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

kubectl apply -f {{ .Project }}`

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
{{if .NewImages -}}
* Copy the source directory into the containers folder for packaging as containers using "copysource.sh <SRC_DIR>"
* Build your images using "buildimages.sh"
* Push images to registry "pushimages.sh <REGISTRY_URL> <REGISTRY_NAMESPACE>"
{{end -}}
{{- if .Helm -}}* Your helm chart is at {{ .Project }}, you can install it using "helminstall.sh" or you can use the operator.{{- else -}}
* Use "deploy.sh" to deploy your artifacts into a kubernetes cluster.
{{- end}}
`

	KnativeReadme_md = `Move2Kube
---------
Congratulations! Move2Kube has generated the necessary build artfiacts for moving all your application components to Knative. Using the artifacts in this directory you can deploy your application in a Knative instance

Prerequisites
-------------
* Docker
* Kubectl

Next Steps
----------
{{if .NewImages -}}
* Copy this directory into your base source directory, so that the scripts gets merged at the right contexts. 
* Build your images using buildimages.sh
* Push images to registry pushimages.sh
{{end -}}
* Use deploy.sh to deploy your artifacts into a knative.`

	Manualimages_md = `Manual containers
-----------------
There is no known automated containerization approach for the below container requirements.

{{range $image := .Images}}{{$image}}
{{end}}`

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