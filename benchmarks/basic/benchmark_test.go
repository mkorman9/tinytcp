package basic

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/mkorman9/tinytcp"
	"io"
	"os"
	"testing"
)

var payload = preparePayload(1024)

func BenchmarkSingleClient(b *testing.B) {
	listener := newMockListener()
	server := createEchoServer(listener)
	defer server.Stop()

	buffer := make([]byte, len(payload))

	b.ResetTimer()

	client := listener.Connect()

	for i := 0; i < b.N; i++ {
		_, err := client.Write(payload)
		if err != nil {
			break
		}

		_, err = client.Read(buffer)
		if err != nil {
			continue
		}
	}
}

func BenchmarkConcurrentClients(b *testing.B) {
	listener := newMockListener()
	server := createEchoServer(listener)
	defer server.Stop()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		client := listener.Connect()
		buffer := make([]byte, len(payload))

		for pb.Next() {
			_, err := client.Write(payload)
			if err != nil {
				break
			}

			_, err = client.Read(buffer)
			if err != nil {
				continue
			}
		}
	})
}

func createEchoServer(listener *mockListener) *tinytcp.Server {
	server := tinytcp.NewServer("fakeaddress")
	server.Listener(listener)

	ch := make(chan struct{})

	server.OnStart(func() {
		ch <- struct{}{}
	})

	server.ForkingStrategy(tinytcp.GoroutinePerConnection(
		tinytcp.PacketFramingHandler(
			tinytcp.LengthPrefixedFraming(tinytcp.PrefixVarInt),
			func(socket *tinytcp.Socket) tinytcp.PacketHandler {
				return func(packet []byte) {
					_, err := socket.Write(packet)
					if err != nil {
						if err == io.EOF {
							return
						}

						fmt.Printf("Error while writing: %v\n", err)
					}
				}
			},
		),
	))

	go func() {
		_ = server.Start()
	}()

	<-ch

	return server
}

func preparePayload(size int) []byte {
	payload := make([]byte, size+1)

	_, err := rand.Read(payload)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		return nil
	}

	var buff bytes.Buffer

	err = tinytcp.WriteVarInt(&buff, len(payload))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		return nil
	}

	_, err = buff.Write(payload)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v", err)
		return nil
	}

	return buff.Bytes()
}
