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

# Build Stage
FROM mcr.microsoft.com/dotnet/framework/sdk:{{ .BaseImageVersion }} As builder
WORKDIR /app
COPY . .
RUN nuget restore

COPY {{ .AppName }}/. ./{{ .AppName }}/
WORKDIR /app/{{ .AppName }}
RUN msbuild /p:Configuration=Release

# Run Stage
FROM microsoft/aspnet:4.7.2 AS runtime
{{- range .Ports}}
EXPOSE {{.}}
{{- end}}
WORKDIR /inetpub/wwwroot
COPY --from=builder /app/{{ .AppName }}/. ./