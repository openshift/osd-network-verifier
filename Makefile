include project.mk

# Include shared Makefiles
include boilerplate/generated-includes.mk

GOFLAGS=-mod=mod
VERSION:=`git describe --tags --abbrev=0`
COMMIT:=`git rev-parse HEAD`
SHORTCOMMIT:=`git rev-parse --short HEAD`
PREFIX=github.com/openshift/osd-network-verifier/version
LDFLAGS=-ldflags="-X '$(PREFIX).Version=$(VERSION)' -X '$(PREFIX).CommitHash=$(COMMIT)' -X '$(PREFIX).ShortCommitHash=$(SHORTCOMMIT)'"

.PHONY: build
build:
	${GOENV} go fmt ./...
	${GOENV} go mod tidy
	${GOENV} go build $(LDFLAGS) $(GOFLAGS) .

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
	${GOENV} go test $(GOFLAGS) ./...

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
