package main

import (
	"fmt"
	"github.com/mkorman9/tinytcp"
	"github.com/mkorman9/tinytcp/promtinytcp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io"
	"net/http"
)

func main() {
	server := tinytcp.NewServer("0.0.0.0:7000")

	server.OnMetricsUpdate(promtinytcp.NewHandler(prometheus.DefaultRegisterer))

	server.ForkingStrategy(tinytcp.GoroutinePerConnection(
		tinytcp.PacketFramingHandler(
			tinytcp.SplitBySeparator([]byte{'\n'}),
			serve,
		),
	))

	go func() {
		http.Handle("/", promhttp.Handler())
		_ = http.ListenAndServe("0.0.0.0:8080", nil)
	}()

	if err := tinytcp.StartAndBlock(server); err != nil {
		fmt.Printf("Error while starting: %v\n", err)
	}
}

func serve(socket *tinytcp.Socket) tinytcp.PacketHandler {
	return func(packet []byte) {
		_, err := socket.Write(packet)
		if err != nil {
			if err == io.EOF {
				return
			}

			fmt.Printf("Error while writing: %v\n", err)
		}
	}
}
