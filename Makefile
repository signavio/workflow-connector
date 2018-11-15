# Binary name
BIN := workflow-connector
# Module name
MODULE := github.com/signavio/workflow-connector
# Which architecture to build - see $(ALL_ARCH) for options.
ARCH ?= amd64
# Which operating system to target - see $(ALL_OS) for options.
OS ?= linux
# Which specific build stage do we want
STAGE ?= production
# This version-strategy uses git tags to set the version string
VERSION := $(shell git describe --tags --always --dirty)
ALL_ARCH := amd64 arm arm64
ALL_OS := linux windows
# If you want to build all binaries, see the 'all-build' rule.
all: build

build: bin/$(ARCH)/$(OS)/$(BIN)

bin/$(ARCH)/$(OS)/$(BIN): build-dirs
	@go build -o bin/$(BIN)_$(ARCH)_$(OS)_$(VERSION)

install: build

version:
	@echo $(VERSION)

build-dirs:
	@mkdir -p bin/$(ARCH)/$(OS)

clean:
	rm -rf bin
