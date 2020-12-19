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

GO_VERSION  ?= 1.15
BINNAME     ?= move2kube
BINDIR      := $(CURDIR)/bin
DISTDIR		:= $(CURDIR)/_dist
TARGETS     := darwin/amd64 linux/amd64
REGISTRYNS  := quay.io/konveyor

GOPATH        = $(shell go env GOPATH)
GOX           = $(GOPATH)/bin/gox
GOLINT        = $(GOPATH)/bin/golint 
GOTEST        = ${GOPATH}/bin/gotest
GOLANGCILINT  = $(GOPATH)/bin/golangci-lint 
GOLANGCOVER   = $(GOPATH)/bin/goveralls 

PKG        := ./...
LDFLAGS    := -w -s

SRC        = $(shell find . -type f -name '*.go' -print0)
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
	VERSION ?= $(BINARY_VERSION)
endif
VERSION ?= latest

VERSION_METADATA = unreleased
ifneq ($(GIT_TAG),)
	VERSION_METADATA =
endif
LDFLAGS += -X github.com/konveyor/${BINNAME}/types/info.buildmetadata=${VERSION_METADATA}

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
	go build -tags excludecodegen,excludedist -ldflags '$(LDFLAGS)' -o $(BINDIR)/$(BINNAME) ./cmd/${BINNAME}
	mkdir -p $(GOPATH)/bin/
	cp $(BINDIR)/$(BINNAME) $(GOPATH)/bin/

.PHONY: build-kubectl-translate
build-kubectl-translate: get $(BINDIR)/kubectl-translate ## Build translate plugin for kubectl https://github.com/kubernetes-sigs/krew

$(BINDIR)/kubectl-translate: $(SRC)
	go build -tags excludecodegen,excludedist -ldflags '$(LDFLAGS)' -o $(BINDIR)/kubectl-translate ./cmd/kubectltranslate

.PHONY: get
get: go.mod
	go mod download

.PHONY: generate
generate:
	go generate ${PKG}

.PHONY: deps
deps: 
	source scripts/installdeps.sh

# -- Test --

.PHONY: test
test: ## Run tests
	go test -run . $(PKG) -race

${GOTEST}:
	${GOGET} github.com/rakyll/gotest

.PHONY: test-verbose
test-verbose: ${GOTEST}
	gotest -run . $(PKG) -race -v

${GOLANGCOVER}:
	${GOGET} github.com/mattn/goveralls@v0.0.6

.PHONY: test-coverage
test-coverage: ${GOLANGCOVER} ## Run tests with coverage
	go test -run . $(PKG) -coverprofile=coverage.txt -covermode=atomic

${GOLANGCILINT}:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.31.0

${GOLINT}:
	${GOGET} golang.org/x/lint/golint

.PHONY: test-style
test-style: ${GOLANGCILINT} ${GOLINT} 
	${GOLANGCILINT} run --timeout 3m
	${GOLINT} ${PKG}
	scripts/licensecheck.sh

# -- CI --

.PHONY: ci
ci: clean build test test-style ## Run CI routine

# -- Release --

$(GOX):
	${GOGET} github.com/mitchellh/gox@v1.0.1

.PHONY: build-cross
build-cross: $(GOX) clean
	CGO_ENABLED=0 $(GOX) -parallel=3 -output="$(DISTDIR)/{{.OS}}-{{.Arch}}/$(BINNAME)" -osarch='$(TARGETS)' -ldflags '$(LDFLAGS)' ./cmd/${BINNAME}

.PHONY: build-cross-kubectl-translate
build-cross-kubectl-translate: $(GOX) clean
	CGO_ENABLED=0 $(GOX) -parallel=3 -output="$(DISTDIR)/{{.OS}}-{{.Arch}}/kubectl-translate" -osarch='$(TARGETS)' -ldflags '$(LDFLAGS)' ./cmd/kubectltranslate

.PHONY: dist
dist: clean build-cross ## Build distribution
	mkdir -p $(DISTDIR)/files
	cp -r ./LICENSE ./scripts/installdeps.sh ./USAGE.md ./samples $(DISTDIR)/files/
	cd $(DISTDIR) && go run ../scripts/builddist.go -b ${BINNAME} -v ${VERSION}

.PHONY: dist-kubectl-translate
dist-kubectl-translate: clean build-cross-kubectl-translate ## Build kubectl plugin distribution
	mkdir -p $(DISTDIR)/files
	cp -r ./LICENSE $(DISTDIR)/files/
	cd $(DISTDIR) && go run ../scripts/builddist.go -b kubectl-translate -v ${VERSION}

.PHONY: clean
clean:
	rm -rf $(BINDIR) $(DISTDIR)
	go clean -cache

.PHONY: info
info: ## Get version info
	 @echo "Version:           ${VERSION}"
	 @echo "Git Tag:           ${GIT_TAG}"
	 @echo "Git Commit:        ${GIT_COMMIT}"
	 @echo "Git Tree State:    ${GIT_DIRTY}"

# -- Docker --

.PHONY: cbuild
cbuild: ## Build docker image
	docker build -t ${REGISTRYNS}/${BINNAME}-builder:${VERSION} --cache-from ${REGISTRYNS}/${BINNAME}-builder:latest --target build_base                          --build-arg VERSION=${VERSION} --build-arg GO_VERSION=${GO_VERSION} .
	docker build -t ${REGISTRYNS}/${BINNAME}:${VERSION}         --cache-from ${REGISTRYNS}/${BINNAME}-builder:latest --cache-from ${REGISTRYNS}/${BINNAME}:latest --build-arg VERSION=${VERSION} --build-arg GO_VERSION=${GO_VERSION} .
	docker tag ${REGISTRYNS}/${BINNAME}-builder:${VERSION} ${REGISTRYNS}/${BINNAME}-builder:latest
	docker tag ${REGISTRYNS}/${BINNAME}:${VERSION} ${REGISTRYNS}/${BINNAME}:latest

.PHONY: cpush
cpush: ## Push docker image
	# To help with reusing layers and hence speeding up build
	docker push ${REGISTRYNS}/${BINNAME}-builder:${VERSION}
	docker push ${REGISTRYNS}/${BINNAME}:${VERSION}
