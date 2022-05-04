default: build

include boilerplate/generated-includes.mk

SHELL := /usr/bin/env bash

# Include shared Makefiles
include project.mk
include standard.mk

# Extend Makefile after here

# Project specific values
IMAGE_REGISTRY ?= quay.io
IMAGE_USER ?= app-sre
IMAGE_NAME ?= osd-network-verifier

DOCKERFILE := ./Dockerfile

# Podman by default, fall back to docker, allow override
CONTAINER_ENGINE ?= $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)

# Gather commit number for Z and short SHA
COMMIT_NUMBER := $(shell git rev-list `git rev-list --parents HEAD | egrep "^[a-f0-9]{40}$$"`..HEAD --count)
CURRENT_COMMIT := $(shell git rev-parse --short=7 HEAD)

# Build container version
VERSION_MAJOR ?= 0
VERSION_MINOR ?= 1
IMAGE_VERSION := $(VERSION_MAJOR).$(VERSION_MINOR).$(COMMIT_NUMBER)-$(CURRENT_COMMIT)

IMAGE_URI_BASE := $(IMAGE_REGISTRY)/$(IMAGE_USER)/$(IMAGE_NAME)
IMAGE_URI_LATEST=$(IMAGE_URI_BASE):latest
IMAGE_URI := $(IMAGE_URI_BASE):v$(IMAGE_VERSION)
# Used by deploy.sh
export IMAGE_URI

GOFLAGS=-mod=mod

.PHONY: build
build:
	go fmt ./...
	go mod tidy
	go build $(GOFLAGS) .

.PHONY: test
test:
	go test $(GOFLAGS) ./...

.PHONY: image-push
image-push:
	${CONTAINER_ENGINE} push $(IMAGE_URI)
	${CONTAINER_ENGINE} push $(IMAGE_URI_LATEST)

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

.PHONY: image-build
image-build: build
	${CONTAINER_ENGINE} build . -f $(DOCKERFILE) -t $(IMAGE_URI)
	${CONTAINER_ENGINE} tag $(IMAGE_URI) $(IMAGE_URI_LATEST)

.PHONY: deploy
deploy:
	hack/deploy.sh
