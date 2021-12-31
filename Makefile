# set default shell
SHELL=/bin/bash -o pipefail -o errexit

TAG ?= $(shell git describe --tags --dirty --always)

REPO_INFO ?= $(shell git config --get remote.origin.url)
COMMIT_SHA ?= git-$(shell git rev-parse --short HEAD)
PKG = github.com/zhengtianbao/nfscp

HOST_ARCH = $(shell which go >/dev/null 2>&1 && go env GOARCH)
ARCH ?= $(HOST_ARCH)
ifeq ($(ARCH),)
    $(error mandatory variable ARCH is empty, either set it when calling the command or make sure 'go env GOARCH' works)
endif

TARGETS_DIR = _output/bin/${ARCH}

.PHONY: build
build:
	CGO_ENABLED=0 go build \
	  -gcflags=all="-N -l" \
      -ldflags="-buildid= \
      -X ${PKG}/version.RELEASE=${TAG} \
      -X ${PKG}/version.COMMIT=${COMMIT_SHA} \
      -X ${PKG}/version.REPO=${REPO_INFO}" \
      -o "${TARGETS_DIR}/nfscp" "${PKG}/cmd/nfscp"

.PHONY: image
image: build
	docker build -t nfscp .

.PHONY: clean
clean:
	rm -rf _output