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


FROM rust:1 as builder
WORKDIR /{{ .AppName }}
COPY . .
RUN cargo build --release

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.3-201
WORKDIR /{{ .AppName }}
COPY --from=builder /{{ .AppName }}/target/release/{{ .AppName }} /{{ .AppName }}/
{{- if .RocketToml}}
COPY --from=builder /{{ .AppName }}/{{ .RocketToml }} /{{ .AppName }}/
ENV ROCKET_ADDRESS={{ .RocketAddress }}
{{- end }}
EXPOSE {{ .Port }}
CMD ["./{{ .AppName }}"]
