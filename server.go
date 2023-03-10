package tinytcp

import (
	"errors"
	"net"
	"sync"
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
	housekeepingJob *housekeepingJob

	errorChannel chan error
	isRunning    bool
	runningMutex sync.Mutex
	abortOnce    sync.Once

	metricsUpdateHandler func(ServerMetrics)
	startHandler         func()
	stopHandler          func()
}

// NewServer returns new Server instance.
func NewServer(address string, config ...*ServerConfig) *Server {
	var providedConfig *ServerConfig
	if config != nil {
		providedConfig = config[0]
	}
	c := mergeServerConfig(providedConfig)

	s := &Server{
		config:               c,
		address:              address,
		listener:             newListener(address, c),
		sockets:              newSocketsList(c.MaxClients),
		errorChannel:         make(chan error, 1),
		metricsUpdateHandler: func(_ ServerMetrics) {},
		startHandler:         func() {},
		stopHandler:          func() {},
	}

	s.housekeepingJob = newHousekeepingJob(c.TickInterval, s.housekeepingJobTick, s.housekeepingJobPanic)

	return s
}

// ForkingStrategy sets forking strategy used by this server (see ForkingStrategy).
func (s *Server) ForkingStrategy(forkingStrategy ForkingStrategy) {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if s.isRunning {
		return
	}

	s.forkingStrategy = forkingStrategy
}

// Listener allows to overwrite the default listener. Should be used with care.
func (s *Server) Listener(listener Listener) {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if s.isRunning {
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

// Start starts TCP server and blocks until Stop() or Abort() are called.
func (s *Server) Start() error {
	err := func() error {
		s.runningMutex.Lock()
		defer s.runningMutex.Unlock()

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

		s.housekeepingJob.Start()
		s.forkingStrategy.OnStart()
		s.startHandler()

		s.isRunning = true
		return nil
	}()

	if err != nil {
		return err
	}

	return s.acceptLoop()
}

// Stop immediately stops the server and unblocks the Start() method.
func (s *Server) Stop() (err error) {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if !s.isRunning {
		return
	}
	s.isRunning = false

	if e := s.listener.Close(); e != nil {
		if !isBrokenPipe(e) {
			err = e
		}
	}

	s.housekeepingJob.Stop()
	s.sockets.Reset()
	s.forkingStrategy.OnStop()
	s.stopHandler()

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

func (s *Server) housekeepingJobTick() {
	s.updateMetrics()
	s.sockets.Cleanup()
}

func (s *Server) housekeepingJobPanic(err error) {
	_ = s.Abort(err)
}

func (s *Server) updateMetrics() {
	var (
		readsPerInterval  uint64
		writesPerInterval uint64
	)

	s.sockets.Iterate(func(socket *Socket) {
		reads, writes := socket.updateMetrics(s.config.TickInterval)
		readsPerInterval += reads
		writesPerInterval += writes
	})

	s.metrics.Connections = s.sockets.Len()
	s.metrics.TotalRead += readsPerInterval
	s.metrics.TotalWritten += writesPerInterval
	s.metrics.ReadLastSecond = uint64(float64(readsPerInterval) / s.config.TickInterval.Seconds())
	s.metrics.WrittenLastSecond = uint64(float64(writesPerInterval) / s.config.TickInterval.Seconds())

	s.forkingStrategy.OnMetricsUpdate(&s.metrics)
	s.metricsUpdateHandler(s.metrics)
}
