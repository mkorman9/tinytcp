package main

import (
	"fmt"
	"github.com/mkorman9/tinytcp"
	"io"
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
	var buffer [4096]byte

	for {
		n, err := socket.Read(buffer[:])
		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Printf("Error while reading: %v", err)
			break
		}

		message := buffer[:n]
		fmt.Printf("Received: %s\n", message)

		_, err = socket.Write(message)
		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Printf("Error while writing: %v", err)
			break
		}
	}
}
