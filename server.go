package tinytcp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Server represents a TCP server. Server is responsible for accepting new connections using Listener,
// and passing them to their respective handlers, defined by given ForkingStrategy.
// This struct conforms to the Service interface.
type Server struct {
	config          *ServerConfig
	address         string
	listener        Listener
	forkingStrategy ForkingStrategy
	sockets         *socketsList
	metrics         ServerMetrics

	errorChannel chan error
	isRunning    int32
	runningMutex sync.Mutex
	ticker       *time.Ticker
	abortOnce    sync.Once

	metricsUpdateHandler func(ServerMetrics)
	startHandler         func()
	stopHandler          func()
	socketPanicHandler   func(error)
	serverPanicHandler   func(error)
	acceptErrorHandler   func(error)
}

// NewServer returns new Server instance.
func NewServer(address string, config ...*ServerConfig) *Server {
	var providedConfig *ServerConfig
	if config != nil {
		providedConfig = config[0]
	}
	c := mergeServerConfig(providedConfig)

	return &Server{
		config:       c,
		address:      address,
		listener:     newListener(address, c),
		sockets:      newSocketsList(c.MaxClients),
		errorChannel: make(chan error, 1),
	}
}

// ForkingStrategy sets forking strategy used by this server (see ForkingStrategy).
func (s *Server) ForkingStrategy(forkingStrategy ForkingStrategy) {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if atomic.LoadInt32(&s.isRunning) == 1 {
		return
	}

	s.forkingStrategy = forkingStrategy
}

// Listener allows to overwrite the default listener. Should be used with care.
func (s *Server) Listener(listener Listener) {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if atomic.LoadInt32(&s.isRunning) == 1 {
		return
	}

	s.listener = listener
}

// Port returns a port number used by underlying Listener. Only returns a valid value after Start().
func (s *Server) Port() int {
	return resolveNetworkPort(s.listener.Addr())
}

// Metrics returns aggregated server metrics.
func (s *Server) Metrics() ServerMetrics {
	return s.metrics
}

// OnMetricsUpdate sets a handler that is called everytime the server metrics are updated.
func (s *Server) OnMetricsUpdate(handler func(ServerMetrics)) {
	s.metricsUpdateHandler = handler
}

// OnStart sets a handler that is called when server starts.
func (s *Server) OnStart(handler func()) {
	s.startHandler = handler
}

// OnStop sets a handler that is called when server stops.
func (s *Server) OnStop(handler func()) {
	s.stopHandler = handler
}

// OnServerPanic sets a handler for panics inside server code.
func (s *Server) OnServerPanic(handler func(error)) {
	s.serverPanicHandler = handler
}

// OnSocketPanic sets a handler for panics inside socket handlers.
func (s *Server) OnSocketPanic(handler func(error)) {
	s.socketPanicHandler = handler
}

// OnAcceptError sets a handler for errors returned by Accept().
func (s *Server) OnAcceptError(handler func(error)) {
	s.acceptErrorHandler = handler
}

// Start starts TCP server and blocks until Stop() or Abort() are called.
func (s *Server) Start() error {
	s.runningMutex.Lock()

	if s.listener == nil {
		return errors.New("empty listener")
	}
	if s.forkingStrategy == nil {
		return errors.New("empty forking strategy")
	}

	err := s.listener.Listen()
	if err != nil {
		return err
	}

	s.startBackgroundJob()
	s.forkingStrategy.OnStart(s.socketPanicHandler)

	if s.startHandler != nil {
		s.startHandler()
	}

	atomic.StoreInt32(&s.isRunning, 1)
	s.runningMutex.Unlock()

	return s.acceptLoop()
}

// Stop immediately stops the server and unblocks the Start() method.
func (s *Server) Stop() (err error) {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if !atomic.CompareAndSwapInt32(&s.isRunning, 1, 0) {
		return
	}

	if e := s.listener.Close(); e != nil {
		if !isBrokenPipe(e) {
			err = e
		}
	}

	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.ticker = nil

	s.sockets.ExecWrite(func(head *Socket) {
		for socket := head; socket != nil; socket = socket.next {
			_ = socket.Close()
			socket.recycle()
		}
	})
	s.sockets.Cleanup()

	s.forkingStrategy.OnStop()

	if s.stopHandler != nil {
		s.stopHandler()
	}

	return
}

// Abort immediately stops the server with error and unblocks the Start() method.
func (s *Server) Abort(e error) (err error) {
	s.abortOnce.Do(func() {
		select {
		case s.errorChannel <- e:
		default:
		}

		err = s.Stop()
	})

	return
}

func (s *Server) acceptLoop() error {
	for {
		connection, err := s.listener.Accept()
		if err != nil {
			if isBrokenPipe(err) {
				break
			}

			if s.acceptErrorHandler != nil {
				s.acceptErrorHandler(err)
			}
			continue
		}

		s.handleNewConnection(connection)
	}

	select {
	case err := <-s.errorChannel:
		return err
	default:
		return nil
	}
}

func (s *Server) handleNewConnection(connection net.Conn) {
	socket := s.sockets.New(connection)
	if socket == nil {
		return
	}

	s.forkingStrategy.OnAccept(socket)
}

func (s *Server) startBackgroundJob() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := errors.New("server background job restart loop")

				if s.serverPanicHandler != nil {
					s.serverPanicHandler(err)
				}
				_ = s.Abort(err)
			}
		}()

		if s.ticker == nil {
			s.ticker = time.NewTicker(s.config.TickInterval)
		}

		for range s.ticker.C {
			s.updateMetrics()
			s.sockets.Cleanup()
		}
	}()
}

func (s *Server) updateMetrics() {
	s.sockets.ExecRead(func(head *Socket) {
		s.metrics.Connections = s.sockets.Len()

		var (
			readsPerInterval  uint64
			writesPerInterval uint64
		)

		for socket := head; socket != nil; socket = socket.next {
			reads, writes := socket.updateMetrics(s.config.TickInterval)
			readsPerInterval += reads
			writesPerInterval += writes
		}

		s.metrics.TotalRead += readsPerInterval
		s.metrics.TotalWritten += writesPerInterval
		s.metrics.ReadLastSecond = uint64(float64(readsPerInterval) / s.config.TickInterval.Seconds())
		s.metrics.WrittenLastSecond = uint64(float64(writesPerInterval) / s.config.TickInterval.Seconds())

		s.forkingStrategy.OnMetricsUpdate(&s.metrics)

		if s.metricsUpdateHandler != nil {
			s.metricsUpdateHandler(s.metrics)
		}
	})
}
