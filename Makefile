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

BIN_DIR=./bin
BIN_NAME=move2kube.wasm
WEB_UI_DIR=m2k-web-ui
GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git tag --points-at | tail -n 1)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

PKG        := ./...
#LDFLAGS    := -w -s
LDFLAGS    :=

VERSION ?= latest

VERSION_METADATA = unreleased
ifneq ($(GIT_TAG),)
	VERSION_METADATA =
endif
LDFLAGS += -X github.com/konveyor/move2kube-wasm/types/info.buildmetadata=${VERSION_METADATA}

LDFLAGS += -X github.com/konveyor/move2kube-wasm/types/info.gitCommit=${GIT_COMMIT}
LDFLAGS += -X github.com/konveyor/move2kube-wasm/types/info.gitTreeState=${GIT_DIRTY}

.PHONY: all
all:
	make clean && make build && make build-web && make serve-web

.PHONY: build
build:
	mkdir -p "${BIN_DIR}"
	CGO_ENABLED=0 GOOS=wasip1 GOARCH=wasm go build -ldflags '$(LDFLAGS)' -o "${BIN_DIR}/${BIN_NAME}" .
	gzip -f "${BIN_DIR}/${BIN_NAME}"
	# We have to put require github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af
	# in order for logrus to work. See https://github.com/HarikrishnanBalagopal/test-wasi-fs-browser/tree/main
	# CGO_ENABLED=0 tinygo build -o "${BIN_DIR}/${BIN_NAME}" -target=wasi .

.PHONY: clean
clean:
	rm -rf "${BIN_DIR}"

.PHONY: run
run:
	wasmer "${BIN_DIR}/${BIN_NAME}"

.PHONY: build-web
build-web:
	cd "${WEB_UI_DIR}" && pnpm run build

.PHONY: serve-web
serve-web:
	cd "${WEB_UI_DIR}" && pnpm run serve

.PHONY: copy-web
copy-web:
	rm -rf docs/
	cp -r "${WEB_UI_DIR}/dist" docs

.PHONY: generate
generate: ## Generate assets
	go generate ${PKG}

.PHONY: dev
dev:
	cd "${WEB_UI_DIR}" && pnpm run dev
