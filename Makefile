.PHONY: build test vet run

build:
	go build -o bin/bookmarks .

test:
	go test ./...

vet:
	go vet ./...

run:
	go run .
