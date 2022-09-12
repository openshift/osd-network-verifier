include project.mk

# Include shared Makefiles
include boilerplate/generated-includes.mk

GOFLAGS=-mod=mod

.PHONY: build
build:
	go fmt ./...
	go mod tidy
	go build $(GOFLAGS) .

.PHONY: fmt
fmt: generate
	go fmt ./...
	go mod tidy

.PHONY: check-fmt
check-fmt: fmt
	git status --porcelain
	@(test 0 -eq $$(git status --porcelain | wc -l)) || (echo "Local git checkout is not clean, commit changes and try again." >&2 && exit 1)

.PHONY: generate
generate:
	go install github.com/golang/mock/mockgen@v1.6.0
	mockgen -source=pkg/cloudclient/aws/aws.go -package mocks -destination=pkg/cloudclient/mocks/mock_aws.go
	mockgen -source=pkg/cloudclient/cloudclient.go -package mocks -destination=pkg/cloudclient/mocks/mock_cloudclient.go

.PHONY: test
test:
	go test $(GOFLAGS) ./...

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
