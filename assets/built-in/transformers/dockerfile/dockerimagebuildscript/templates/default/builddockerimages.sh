#!/usr/bin/env bash
#   Copyright IBM Corporation 2021
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

if [[ "$(basename "$PWD")" != 'scripts' ]] ; then
  echo 'please run this script from the "scripts" directory'
  exit 1
fi

cd .. # go to the parent directory so that all the relative paths will be correct
{{- if .DockerBuild }}
{{- range $dockerfile := .DockerfilesConfig }}

echo 'building image {{ $dockerfile.ImageName }}'
cd {{ $dockerfile.ContextUnix }}
docker build -f {{ $dockerfile.DockerfileName }} -t {{ $dockerfile.ImageName }} .
cd -
{{- end }}
{{- else if .PodmanBuild }}
{{- range $dockerfile := .DockerfilesConfig }}

echo 'building image {{ $dockerfile.ImageName }}'
cd {{ $dockerfile.ContextUnix }}
podman build -f {{ $dockerfile.DockerfileName }} -t {{ $dockerfile.ImageName }} .
cd -
{{- end }}
{{ else if .BuildxBuild }}
if [ "$#" -ne 3 ]; then
    REGISTRY_URL={{ .RegistryURL }}
    REGISTRY_NAMESPACE={{ .RegistryNamespace }}
    PLATFORMS={{ .TargetPlatforms }}
else
    REGISTRY_URL=$1
    REGISTRY_NAMESPACE=$2
    PLATFORMS=$3
fi
{{- range $dockerfile := .DockerfilesConfig }}

echo 'building image {{ $dockerfile.ImageName }}'
cd {{ $dockerfile.ContextUnix }}
docker buildx build --platform ${PLATFORMS} -f {{ $dockerfile.DockerfileName }}  --push --tag ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/{{ $dockerfile.ImageName }} .
cd -
{{- end }}
{{- end }}
echo 'done'
