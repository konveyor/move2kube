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
FROM --platform=windows/amd64 mcr.microsoft.com/dotnet/framework/sdk:{{ .BaseImageVersion }} As builder
WORKDIR /app
COPY . .
RUN msbuild /p:Configuration=Release /p:OutputPath=/app/output

# Run Stage
FROM --platform=windows/amd64 mcr.microsoft.com/dotnet/framework/runtime:{{ .BaseImageVersion }}
{{- range $port := .Ports }}
EXPOSE {{ $port }}
{{- end }}
WORKDIR /app
COPY --from=builder /app/output/ .
CMD {{ .AppName }}.exe