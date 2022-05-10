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

.PHONY: fmt
fmt:
	go fmt ./...
	go mod tidy

.PHONY: check-fmt
check-fmt: fmt
	git status --porcelain
	@(test 0 -eq $$(git status --porcelain | wc -l)) || (echo "Local git checkout is not clean, commit changes and try again." >&2 && exit 1)

.PHONY: test
test:
	go test $(GOFLAGS) ./...

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update



