#!/usr/bin/env bash
#   Copyright IBM Corporation 2020
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
{{ $containerRuntime := .ContainerRuntime }}

REGISTRY_URL={{ .RegistryURL }}
REGISTRY_NAMESPACE={{ .RegistryNamespace }}
if [ "$#" -eq 2 ]; then
    REGISTRY_URL=$1
    REGISTRY_NAMESPACE=$2
fi

# Uncomment the below line if you want to enable login before pushing
# {{ $containerRuntime }} login ${REGISTRY_URL}
{{- range $image := .Images }}

echo 'pushing image {{ $image }}'
{{ $containerRuntime }} tag {{ $image }} ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/{{ $image }}
{{ $containerRuntime }} push ${REGISTRY_URL}/${REGISTRY_NAMESPACE}/{{ $image }}
{{- end }}

echo 'done'
