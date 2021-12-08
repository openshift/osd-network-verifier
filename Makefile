GOFLAGS=-mod=mod

.PHONY: build
build: 
	$(GOFLAGS) go fmt ./...
	$(GOFLAGS) go mod tidy
	$(GOFLAGS) go build .
test: 
	$(GOFLAGS) go test 
