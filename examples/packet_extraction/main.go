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

	server.ForkingStrategy(tinytcp.GoroutinePerConnection(
		tinytcp.PacketFramingHandler(
			tinytcp.SplitBySeparator([]byte{'\n'}),
			serve,
		),
	))

	if err := tinytcp.StartAndBlock(server); err != nil {
		fmt.Printf("Error while starting: %v\n", err)
	}
}

func serve(socket *tinytcp.Socket) tinytcp.PacketHandler {
	// client connected

	return func(packet []byte) {
		_, err := socket.Write(packet)
		if err != nil {
			if socket.IsClosed() {
				return
			}
		}
	}
}
