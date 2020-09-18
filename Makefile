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

BINNAME     ?= move2kube
BINDIR      := $(CURDIR)/bin
DISTDIR		:= $(CURDIR)/_dist
TARGETS     := darwin/amd64 linux/amd64

GOPATH        = $(shell go env GOPATH)
GOX           = $(GOPATH)/bin/gox
GOLINT        = $(GOPATH)/bin/golint 
GOTEST        = ${GOPATH}/bin/gotest
GOLANGCILINT  = $(GOPATH)/bin/golangci-lint 
GOLANGCOVER   = $(GOPATH)/bin/goveralls 

PKG        := ./...
LDFLAGS    := -w -s

SRC        = $(shell find . -type f -name '*.go' -print)
ARCH       = $(shell uname -p)
GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

GOGET     := cd / && GO111MODULE=on go get -u 

ifdef VERSION
	BINARY_VERSION = $(VERSION)
endif
BINARY_VERSION ?= ${GIT_TAG}
ifneq ($(BINARY_VERSION),)
	LDFLAGS += -X github.com/konveyor/${BINNAME}/types/info.version=${BINARY_VERSION}
	VERSION = $(BINARY_VERSION)
endif

VERSION_METADATA = unreleased
ifneq ($(GIT_TAG),)
	VERSION_METADATA =
endif
LDFLAGS += -X github.com/konveyor/${BINNAME}/types/info.metadata=${VERSION_METADATA}

LDFLAGS += -X github.com/konveyor/${BINNAME}/types/info.gitCommit=${GIT_COMMIT}
LDFLAGS += -X github.com/konveyor/${BINNAME}/types/info.gitTreeState=${GIT_DIRTY}
LDFLAGS += -extldflags "-static"

# HELP
# This will output the help for each task
.PHONY: help
help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[0-9a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# -- Build --

.PHONY: build
build: get $(BINDIR)/$(BINNAME) ## Build go code

$(BINDIR)/$(BINNAME): $(SRC)
	@go build -tags excludecodegen -ldflags '$(LDFLAGS)' -o $(BINDIR)/$(BINNAME) ./cmd/${BINNAME}
	@cp $(BINDIR)/$(BINNAME) $(GOPATH)/bin/

.PHONY: get
get: go.mod
	@go mod download

.PHONY: generate
generate: 
	@go generate ${PKG}

.PHONY: deps
deps: 
	@source scripts/installdeps.sh

# -- Test --

.PHONY: test
test: ## Run tests
	@go test -run . $(PKG) -race

${GOTEST}:
	${GOGET} github.com/rakyll/gotest

.PHONY: test-verbose
test-verbose: ${GOTEST}
	@gotest -run . $(PKG) -race -v

${GOLANGCOVER}:
	${GOGET} github.com/mattn/goveralls@v0.0.6

.PHONY: test-coverage
test-coverage: ${GOLANGCOVER} ## Run tests with coverage
	@go test -run . $(PKG) -covermode=atomic

${GOLANGCILINT}:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.30.0

${GOLINT}:
	${GOGET} golang.org/x/lint/golint

.PHONY: test-style
test-style: ${GOLANGCILINT} ${GOLINT} 
	${GOLANGCILINT} run
	${GOLINT} ${PKG}
	scripts/licensecheck.sh

# -- Release --

$(GOX):
	${GOGET} github.com/mitchellh/gox@v1.0.1

.PHONY: build-cross
build-cross: $(GOX) clean
	CGO_ENABLED=0 $(GOX) -parallel=3 -output="$(DISTDIR)/{{.OS}}-{{.Arch}}/$(BINNAME)" -osarch='$(TARGETS)' -ldflags '$(LDFLAGS)' ./cmd/${BINNAME}

.PHONY: dist
dist: clean build-cross ## Build Distribution
	@mkdir -p $(DISTDIR)/files
	@cp -r ./{LICENSE,scripts/installdeps.sh,USAGE.md,samples} $(DISTDIR)/files/
	@cd $(DISTDIR) && \
	 find * -maxdepth 1 -name "*-*" -type d \
	  -exec cp -r $(DISTDIR)/files/* {} \; \
	  -exec tar -zcf ${BINNAME}-${VERSION}-{}.tar.gz {} \; \
	  -exec sh -c 'shasum -a 256 ${BINNAME}-${VERSION}-{}.tar.gz > ${BINNAME}-${VERSION}-{}.tar.gz.sha256sum' \; \
	  -exec zip -r ${BINNAME}-${VERSION}-{}.zip {} \; \
	  -exec sh -c 'shasum -a 256 ${BINNAME}-${VERSION}-{}.zip > ${BINNAME}-${VERSION}-{}.zip.sha256sum' \;

.PHONY: clean
clean:
	@rm -rf $(BINDIR) $(DISTDIR)
	@go clean -cache

.PHONY: info
info: ## Get version info
	 @echo "Version:           ${VERSION}"
	 @echo "Git Tag:           ${GIT_TAG}"
	 @echo "Git Commit:        ${GIT_COMMIT}"
	 @echo "Git Tree State:    ${GIT_DIRTY}"

# -- Docker --

.PHONY: cbuild
cbuild: ## Build docker image
	@docker build -t ${BINNAME}:latest .
