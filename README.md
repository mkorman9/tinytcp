# tinytcp

![master status](https://github.com/mkorman9/tinytcp/actions/workflows/master.yml/badge.svg)

tinytcp is a high-performance, zero-allocation TCP server in Go.
It wraps around the standard `net` library to provide a sane API for quick prototyping.
Major features include:

- No external dependencies.
- No memory allocations on critical paths, all the important objects and buffers are pooled.
- Automated packet extraction with no memory allocations (see `LengthPrefixedFraming` or `SplitBySeparator`).
- Full customization of connection handling process. By default, the sever starts a new goroutine for each connection,
(`GoroutinePerConnection` strategy), but this can be changed.
- Metrics collection for both the server and each connected client separately (optional Prometheus binding).
- Support for `tcp`, `tcp4`, `tcp6` and `unix` listeners.

## Install

```bash
go get github.com/mkorman9/tinytcp
```

## Example

```go
package main

import (
	"fmt"
	"github.com/mkorman9/tinytcp"
)

func main() {
	server := tinytcp.NewServer("0.0.0.0:7000")

	server.OnStart(func() {
		fmt.Printf("Server started on: %d\n", server.Port())
	})

	server.ForkingStrategy(tinytcp.GoroutinePerConnection(serve))

	if err := tinytcp.StartAndBlock(server); err != nil {
		fmt.Printf("Error while starting: %v\n", err)
	}
}

func serve(socket *tinytcp.Socket) {
	socket.Write([]byte("Hello world!"))
}
```

## Benchmarks

GOMAXPROCS=1
```
$ GOMAXPROCS=1 go test ./... -bench=. -benchmem
BenchmarkSingleClient             728698              1582 ns/op               0 B/op          0 allocs/op
BenchmarkConcurrentClients        757249              1577 ns/op               0 B/op          0 allocs/op
```

GOMAXPROCS=8
```
$ GOMAXPROCS=8 go test ./... -bench=. -benchmem
BenchmarkSingleClient-8           615766              1838 ns/op               0 B/op          0 allocs/op
BenchmarkConcurrentClients-8     4523785               273.7 ns/op             0 B/op          0 allocs/op
```
