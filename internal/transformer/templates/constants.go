// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots at
// 2020-11-26 22:17:13.42343 +0530 IST m=+0.002183475

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
    echo 'Usage: copysources.sh path/to/sources/'
    echo 'Example: copysources.sh {{ .RelRootDir }}'
}

if [ "$#" -ne 1 ]; then
    print_usage "$@"
    exit 1
fi

cp -r "$1"/* {{.Dst}}/
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
{{if .NewImages -}}
* Copy the source directory into the containers folder for packaging as containers using "copysource.sh <SRC_DIR>"
* Build your images using "buildimages.sh"
* Push images to registry "pushimages.sh <REGISTRY_URL> <REGISTRY_NAMESPACE>"
{{end -}}
{{- if .Helm -}}* Your helm chart is at {{ .Project }}, you can install it using "helminstall.sh" or you can use the operator.{{- else -}}
* Use "deploy.sh" to deploy your artifacts into a kubernetes cluster.
{{- end}}
{{if .AddCopySourcesWarning }}
IMPORTANT!!: If you used the UI for translation then "copysources.sh" may copy to the wrong folder.  
This is a limitation of the beta version. It should be fixed in future versions of move2kube-ui.  
In the meantime you can either:
- copy the sources to the appropriate folders inside "./containers/" manually. "./containers/" has  
  the same folder structure as the sources folder so simply copy the appropriate source files/folders  
  to the corresponding folders inside "./containers/".
- move the sources into a directory with the same name and then try copysources with that.  
  Example: if sources is a folder called "foo" you might try moving it into "foo/foo/foo" and then  
  doing "./copysources.sh path/to/foo/". This will require you to read "copysource.sh" and have some knowledge  
  of how "cp -r" works in order to get it right.
{{ end }}`

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