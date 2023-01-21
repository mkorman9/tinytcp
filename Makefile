.DEFAULT_GOAL := all

test:
	go test -v

benchmark:
	cd benchmarks && GOMAXPROCS=1 go test ./... -bench=. -benchmem

all: test
