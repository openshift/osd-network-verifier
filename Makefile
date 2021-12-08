SHELL := /usr/bin/env bash

default: build

# Include shared Makefiles
include project.mk
include standard.mk

# Extend Makefile after here

GOFLAGS=-mod=mod

.PHONY: build
build:
	$(GOFLAGS) go fmt ./...
	$(GOFLAGS) go mod tidy
	$(GOFLAGS) go build .

.PHONY: test
test: 
	$(GOFLAGS) go test 

.PHONY: build-push
build-push:
	hack/app_sre_build_push.sh $(IMAGE_URI_VERSION)

.PHONY: skopeo-push
skopeo-push: containerbuild
	skopeo copy \
		--dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
		"docker-daemon:${IMAGE_URI_VERSION}" \
		"docker://${IMAGE_URI_VERSION}"
	skopeo copy \
		--dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
		"docker-daemon:${IMAGE_URI_LATEST}" \
		"docker://${IMAGE_URI_LATEST}"
