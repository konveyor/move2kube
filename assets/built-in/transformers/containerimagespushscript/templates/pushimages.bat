:: Copyright IBM Corporation 2021
::
::  Licensed under the Apache License, Version 2.0 (the "License");
::  you may not use this file except in compliance with the License.
::  You may obtain a copy of the License at
::
::        http://www.apache.org/licenses/LICENSE-2.0
::
::  Unless required by applicable law or agreed to in writing, software
::  distributed under the License is distributed on an "AS IS" BASIS,
::  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
::  See the License for the specific language governing permissions and
::  limitations under the License.

:: Invoke as pushimages.bat <registry_url> <registry_namespace> <container_runtime>

@echo off
IF "%3"=="" GOTO DEFAULT_CONTAINER_RUNTIME
SET CONTAINER_RUNTIME=%3%
GOTO :REGISTRY

:DEFAULT_CONTAINER_RUNTIME
    SET CONTAINER_RUNTIME=docker

:REGISTRY
    IF "%2"=="" GOTO DEFAULT_REGISTRY
    IF "%1"=="" GOTO DEFAULT_REGISTRY
    SET REGISTRY_URL=%1
    SET REGISTRY_NAMESPACE=%2
    GOTO :MAIN

:DEFAULT_REGISTRY
    SET REGISTRY_URL={{ .RegistryURL }}
    SET REGISTRY_NAMESPACE={{ .RegistryNamespace }}

:UNSUPPORTED_BUILD_SYSTEM
    echo 'Unsupported build system passed as an argument for pushing the images.'
    GOTO SKIP

:MAIN
IF NOT %CONTAINER_RUNTIME% == "docker" IF NOT %CONTAINER_RUNTIME% == "podman" GOTO UNSUPPORTED_BUILD_SYSTEM
:: Uncomment the below line if you want to enable login before pushing
:: %CONTAINER_RUNTIME% login %REGISTRY_URL%
{{- range $image := .Images }}

echo "pushing image {{ $image }}"
%CONTAINER_RUNTIME% tag {{ $image }} %REGISTRY_URL%/%REGISTRY_NAMESPACE%/{{ $image }}
%CONTAINER_RUNTIME% push %REGISTRY_URL%/%REGISTRY_NAMESPACE%/{{ $image }}
{{- end }}

echo "done"

:SKIP
