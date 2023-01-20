.DEFAULT_GOAL := all

test:
	go test -v ./...

all: test
