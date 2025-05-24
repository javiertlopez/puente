all: test

dependencies:
	go get -v -t -d ./...

build:
	go build -v ./...

fmt:
	go fmt ./...

test:
	go test -v ./...

.PHONY: all dependencies build fmt test