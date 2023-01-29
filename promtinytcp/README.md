Prometheus metrics collector.

## Example

```go
package main

import (
	"fmt"
	"github.com/mkorman9/tinytcp"
	"github.com/mkorman9/tinytcp/promtinytcp"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	server := tinytcp.NewServer("0.0.0.0:7000")

	// create prometheus registry and connect it to the server
	registry := prometheus.NewRegistry()
	server.OnMetricsUpdate(promtinytcp.NewHandler(registry))

	server.ForkingStrategy(tinytcp.GoroutinePerConnection(serve))

	if err := tinytcp.StartAndBlock(server); err != nil {
		fmt.Printf("Error while starting: %v\n", err)
	}
}

func serve(socket *tinytcp.Socket) {
	socket.Write([]byte("Hello world!"))
}
```
