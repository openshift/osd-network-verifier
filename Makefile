default: build
IMAGE_NAME?=osd-network-verifier

# Include shared Makefiles
include boilerplate/generated-includes.mk

# REMOVE FOLLOWING AFTER OSD-11306 IS MERGED
include hack/project.mk
include hack/standard.mk
.PHONY: build-push
build-push:
	hack/app_sre_build_push.sh $(IMAGE_URI_VERSION)
# END REMOVE


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



