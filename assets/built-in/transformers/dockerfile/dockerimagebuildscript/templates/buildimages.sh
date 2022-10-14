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
CONTAINER_RUNTIME=docker
if [ "$#" -eq 1 ]; then
    CONTAINER_RUNTIME=$1
fi
if [ "${CONTAINER_RUNTIME}" != "docker" ] && [ "${CONTAINER_RUNTIME}" != "podman" ]; then
   echo 'Unsupported container runtime passed as an argument for building the images: '"${CONTAINER_RUNTIME}"
   exit 1
fi
cd {{ .RelParentOfSourceDir }} # go to the parent directory so that all the relative paths will be correct

{{- range $dockerfile := .DockerfilesConfig }}

echo 'building image {{ $dockerfile.ImageName }}'
cd {{ $dockerfile.ContextUnix }}
${CONTAINER_RUNTIME} build -f {{ $dockerfile.DockerfileName }} -t {{ $dockerfile.ImageName }} .
cd -
{{- end }}

echo 'done'
