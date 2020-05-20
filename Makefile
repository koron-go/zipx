.PHONY: build
build:
	go build -gcflags '-e'

.PHONY: test
test:
	go test ./...

.PHONY: tags
tags:
	gotags -f tags -R .

.PHONY: cover
cover:
	mkdir -p tmp
	go test -coverprofile tmp/cover.out ./...
	go tool cover -html tmp/cover.out -o tmp/cover.html

.PHONY: lint
lint:
	golint ./...

.PHONY: clean
clean:
	go clean
	rm -f tags

# based on: github.com/koron-go/_skeleton/Makefile
