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

cd {{ .RelParentOfSourceDir }} # go to the parent directory so that all the relative paths will be correct

REGISTRY_URL={{ .RegistryURL }}
REGISTRY_NAMESPACE={{ .RegistryNamespace }}
PLATFORMS="linux/amd64,linux/arm64,linux/s390x,linux/ppc64le"
if [ "$#" -eq 2 ]; then
  REGISTRY_URL=$1
  REGISTRY_NAMESPACE=$2
elif [ "$#" -eq 3 ]; then
  REGISTRY_URL=$1
  REGISTRY_NAMESPACE=$2
  PLATFORMS=$3
fi
# Uncomment the below line if you want to enable login before pushing
# docker login ${REGISTRY_URL}
{{- range $dockerfile := .DockerfilesConfig }}

echo 'building and pushing image {{ $dockerfile.ImageName }}'
cd {{ $dockerfile.ContextUnix }}
docker buildx build --platform ${PLATFORMS} -f {{ $dockerfile.DockerfileName }}  --push --tag ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/{{ $dockerfile.ImageName }} .
cd -
{{- end }}

echo 'done'
