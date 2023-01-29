package tinytcp

import (
	"fmt"
	"sync/atomic"
)

// ForkingStrategy defines the way new connections are handled by the associated TCP server.
// Most naive implementation is to start a new goroutine for each new connection,
// and make this goroutine responsible for the whole lifecycle of the connection.
// This implementation might not fit the needs of some highly-concurrent servers,
// so other implementations (like worker pool) may be implemented on top of this interface.
type ForkingStrategy interface {
	// OnStart is called once, after server start.
	OnStart(panicHandler func(error))

	// OnAccept is called for every connection accepted by the server.
	// The implementation should handle all the interactions with the socket,
	// closing it after use and recovering from any potential panic.
	OnAccept(socket *Socket)

	// OnMetricsUpdate is called every time the server updates its metrics.
	OnMetricsUpdate(metrics *ServerMetrics)

	// OnStop is called once, after server stops.
	OnStop()
}

/*
	Goroutine Per Connection
*/

type goroutinePerConnection struct {
	handler      SocketHandler
	goroutines   int32
	panicHandler func(error)
}

func (g *goroutinePerConnection) OnStart(panicHandler func(error)) {
	g.panicHandler = panicHandler
}

func (g *goroutinePerConnection) OnStop() {
}

func (g *goroutinePerConnection) OnMetricsUpdate(metrics *ServerMetrics) {
	metrics.Goroutines = int(atomic.LoadInt32(&g.goroutines))
}

func (g *goroutinePerConnection) OnAccept(socket *Socket) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if g.panicHandler != nil {
					g.panicHandler(fmt.Errorf("%v", r))
				}
			}
		}()

		defer func() {
			_ = socket.Close()
			socket.recycle()
			atomic.AddInt32(&g.goroutines, -1)
		}()

		atomic.AddInt32(&g.goroutines, 1)

		if g.handler != nil {
			g.handler(socket)
		}
	}()
}

// GoroutinePerConnection is the most naive implementation of the ForkingStrategy.
// This is the recommended implementation for most of the general-purpose TCP servers.
// It starts a new goroutine for every new connection. The handler associated with the connection will be responsible
// for handling blocking operations on this connection.
// Connections are automatically closed after their handler finishes.
func GoroutinePerConnection(handler SocketHandler) ForkingStrategy {
	return &goroutinePerConnection{
		handler: handler,
	}
}
