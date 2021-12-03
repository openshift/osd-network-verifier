SHELL := /usr/bin/env bash

default: build containerbuild

# Include shared Makefiles
include project.mk
include standard.mk

# Extend Makefile after here

.PHONY: build
build: 
	go fmt ./...
	go mod tidy
	go build .

.PHONY: test
test: 
	go test 

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
