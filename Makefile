default: build
IMAGE_NAME?=osd-network-verifier

# Include shared Makefiles
include boilerplate/generated-includes.mk

GOFLAGS=-mod=mod

.PHONY: build
build:
	go fmt ./...
	go mod tidy
	go build $(GOFLAGS) .

.PHONY: test
test:
	go test $(GOFLAGS) ./...

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
