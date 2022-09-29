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

:: Invoke as pushimages.bat <registry_url> <registry_namespace>

@echo off
IF "%2"=="" GOTO DEFAULT
IF "%1"=="" GOTO DEFAULT
    SET REGISTRY_URL=%1
    SET REGISTRY_NAMESPACE=%2
GOTO :MAIN

:DEFAULT
    SET REGISTRY_URL={{ .RegistryURL }}
    SET REGISTRY_NAMESPACE={{ .RegistryNamespace }}

:MAIN
{{- if .DockerBuild }}
:: Uncomment the below line if you want to enable login before pushing
:: docker login %REGISTRY_URL%
{{range $image := .Images}}docker tag {{$image}} %REGISTRY_URL%/%REGISTRY_NAMESPACE%/{{$image}}
docker push %REGISTRY_URL%/%REGISTRY_NAMESPACE%/{{$image}}
{{ end }}
{{- else if .PodmanBuild }}
:: podman login %REGISTRY_URL%

{{range $image := .Images}}podman tag {{$image}} %REGISTRY_URL%/%REGISTRY_NAMESPACE%/{{$image}}
podman push %REGISTRY_URL%/%REGISTRY_NAMESPACE%/{{$image}}
{{ end }}
{{- end }}
