:: Copyright IBM Corporation 2021
::
::  Licensed under the Apache License, Version 2.0 (the "License");
::   you may not use this file except in compliance with the License.
::   You may obtain a copy of the License at
::
::        http://www.apache.org/licenses/LICENSE-2.0
::
::  Unless required by applicable law or agreed to in writing, software
::  distributed under the License is distributed on an "AS IS" BASIS,
::  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
::  See the License for the specific language governing permissions and
::  limitations under the License.

for /F "delims=" %%i in ("%cd%") do set basename="%%~ni"

if not %basename% == "scripts" (
    echo "please run this script from the 'scripts' directory"
    exit 1
)

REM go to the source directory so that all the relative paths will be correct
cd {{ .RelSourceDir }}

{{- range $dockerfile := .Dockerfiles }}

pushd {{ $dockerfile.ContextWindows }}
{{ $dockerfile.ContainerRuntime }} build -f {{ $dockerfile.DockerfileName }} -t {{ $dockerfile.ImageName }} .
popd
{{- end }}

echo "done"
