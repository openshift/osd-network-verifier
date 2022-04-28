include boilerplate/generated-includes.mk

SHELL := /usr/bin/env bash

default: build

# Include shared Makefiles
include project.mk
include standard.mk

# Extend Makefile after here

GOFLAGS=-mod=mod

.PHONY: build
build:
	go fmt ./...
	go mod tidy
	go build $(GOFLAGS) .

.PHONY: test
test:
	go test $(GOFLAGS) ./...

.PHONY: build-push
build-push:
	hack/app_sre_build_push.sh $(IMAGE_URI_VERSION)

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
