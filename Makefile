# set default shell
SHELL=/bin/bash -o pipefail -o errexit

REPO_INFO ?= $(shell git config --get remote.origin.url)
COMMIT_SHA ?= git-$(shell git rev-parse --short HEAD)
PKG = github.com/zhengtianbao/nfscp

HOST_ARCH = $(shell which go >/dev/null 2>&1 && go env GOARCH)
ARCH ?= $(HOST_ARCH)
ifeq ($(ARCH),)
    $(error mandatory variable ARCH is empty, either set it when calling the command or make sure 'go env GOARCH' works)
endif

TARGETS_DIR="_output/bin/${ARCH}"

.PHONY: build
build:  ## Build ingress controller, debug tool and pre-stop hook.
	go build \
      -trimpath -ldflags="-buildid= -w -s \
      -X ${PKG}/version.RELEASE=${TAG} \
      -X ${PKG}/version.COMMIT=${COMMIT_SHA} \
      -X ${PKG}/version.REPO=${REPO_INFO}" \
      -o "${TARGETS_DIR}/nfscp" "${PKG}/cmd/nfscp"

.PHONY: clean
clean: ## Remove .gocache directory.
	rm -rf _output