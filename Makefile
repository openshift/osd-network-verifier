.PHONY: build
build: 
	go fmt ./...
	go mod tidy
	go build .
test: 
	go test 
