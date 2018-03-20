# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# The binary to build (just the basename).
BIN := wfadb

# This repo's root import path (under GOPATH).
PKG := github.com/signavio/workflow-connector

# Where to push the docker image.
REGISTRY ?= sdaros

# Which architecture to build - see $(ALL_ARCH) for options.
ARCH ?= amd64

# Which architecture to build - see $(ALL_OS) for options.
OS ?= linux

# Which specific build stage do we want
STAGE ?= production

# This version-strategy uses git tags to set the version string
#VERSION := $(shell git describe --tags --always --dirty)
#
# This version-strategy uses a manual value to set the version string
VERSION := 0.1.0

###
### These variables should not need tweaking.
###

SRC_DIRS := cmd pkg # directories which hold app source (not vendored) 

ALL_ARCH := amd64 arm arm64 ppc64le
ALL_OS := linux windows

# Set gefault base image dynamically for each arch
ifeq ($(ARCH),amd64)
    BASEIMAGE?=alpine
endif
ifeq ($(ARCH),arm)
    BASEIMAGE?=armel/busybox
endif
ifeq ($(ARCH),arm64)
    BASEIMAGE?=aarch64/busybox
endif
ifeq ($(ARCH),ppc64le)
    BASEIMAGE?=ppc64le/busybox
endif

IMAGE := $(REGISTRY)/$(BIN)-$(ARCH)-$(OS)

BUILD_IMAGE ?= golang:1.9.2-alpine3.6
# If you want to build all binaries, see the 'all-build' rule.
# If you want to build all containers, see the 'all-container' rule.
# If you want to build AND push all containers, see the 'all-push' rule.
all: build

build: bin/$(ARCH)/$(OS)/$(BIN)

bin/$(ARCH)/$(OS)/$(BIN): build-dirs container-testing
	@echo "building: $@"
	@docker run                                                             \
	    -ti                                                                 \
	    --rm                                                                \
	    -u $$(id -u):$$(id -g)                                              \
      -v "$$(pwd)/.go:/go"                                                \
	    -v "$$(pwd):/go/src/$(PKG)"                                         \
	    -v "$$(pwd)/bin/$(ARCH)/$(OS):/go/bin"                                    \
	    -v "$$(pwd)/bin/$(ARCH)/$(OS):/go/bin/$(OS)_$(ARCH)"            \
	    -v "$$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/$(OS)_$(ARCH)_static" \
	    -w /go/src/$(PKG)                                                   \
      $(IMAGE):$(VERSION)-testing                                         \
      /bin/sh -c "                                                        \
          ARCH=$(ARCH)                                                    \
          OS=$(OS)                                                        \
	        VERSION=$(VERSION)                                              \
	        PKG=$(PKG)                                                      \
	        build/build.sh $(SRC_DIRS)                                      \
	    "

install: build container-production

# Example: make shell CMD="-c 'date > datefile'"
shell: build-dirs
	@echo "launching a shell in the containerized build environment"
	@docker run                                                             \
	    -ti                                                                 \
	    --rm                                                                \
	    -u $$(id -u):$$(id -g)                                              \
	    -v "$$(pwd)/.go:/go"                                                \
	    -v "$$(pwd):/go/src/$(PKG)"                                         \
	    -v "$$(pwd)/bin/$(ARCH):/go/bin"                                    \
	    -v "$$(pwd)/bin/$(ARCH):/go/bin/$$(go env GOOS)_$(ARCH)"            \
	    -v "$$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static" \
	    -w /go/src/$(PKG)                                                   \
	    $(BUILD_IMAGE)                                                      \
	    /bin/sh $(CMD)

DOTFILE_IMAGE = $(subst :,_,$(subst /,_,$(IMAGE))-$(VERSION))

container-testing: .container-$(DOTFILE_IMAGE) container-name
	@docker build --target testing -t $(IMAGE):$(VERSION)-testing -f .dockerfile-$(ARCH) .
	@docker images -q $(IMAGE):$(VERSION)-$* > .container-$(DOTFILE_IMAGE)-testing

container-production: .container-$(DOTFILE_IMAGE) container-name
	@docker build --target production -t $(IMAGE):$(VERSION)-production -f .dockerfile-$(ARCH) .
	@docker images -q $(IMAGE):$(VERSION)-$* > .container-$(DOTFILE_IMAGE)-production

.container-$(DOTFILE_IMAGE): Dockerfile.in
	@sed \
	    -e 's|ARG_BIN|$(BIN)|g' \
	    -e 's|ARG_ARCH|$(ARCH)|g' \
	    -e 's|ARG_BASEIMAGE|$(BASEIMAGE)|g' \
	    -e 's|ARG_BUILDIMAGE|$(BUILD_IMAGE)|g' \
	    -e 's|ARG_PKG|$(PKG)|g' \
	    Dockerfile.in > .dockerfile-$(ARCH)

container-name:
	@echo "create container: $(IMAGE):$(VERSION)"

push: .push-$(DOTFILE_IMAGE) push-name
.push-$(DOTFILE_IMAGE): .container-$(DOTFILE_IMAGE)
ifeq ($(findstring gcr.io,$(REGISTRY)),gcr.io)
	@gcloud docker -- push $(IMAGE):$(VERSION)
else
	@docker push $(IMAGE):$(VERSION)
endif
	@docker images -q $(IMAGE):$(VERSION) > $@

push-name:
	@echo "pushed: $(IMAGE):$(VERSION)"

version:
	@echo $(VERSION)

test: build-dirs
	@docker run                                                             \
	    -ti                                                                 \
	    --rm                                                                \
	    -u $$(id -u):$$(id -g)                                              \
	    -v "$$(pwd)/.go:/go"                                                \
	    -v "$$(pwd):/go/src/$(PKG)"                                         \
	    -v "$$(pwd)/bin/$(ARCH):/go/bin"                                    \
	    -v "$$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static" \
	    -w /go/src/$(PKG)                                                   \
      $(IMAGE):$(VERSION)-testing                                         \
	    /bin/sh -c "                                                        \
	        ./build/test.sh $(SRC_DIRS)                                     \
	    "

build-dirs:
	@mkdir -p bin/$(ARCH)/$(OS)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(ARCH)

clean: container-clean bin-clean

container-clean:
	rm -rf .container-* .dockerfile-* .push-*

bin-clean:
	rm -rf .go bin
