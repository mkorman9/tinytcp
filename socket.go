package tinytcp

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Socket represents a connected TCP socket.
// An instance of Socket is only valid inside its designated handler and cannot be stored outside (see SocketRef).
type Socket struct {
	remoteAddr    string
	timestamp     int64
	conn          net.Conn
	reader        io.Reader
	writer        io.Writer
	meteredReader *meteredReader
	meteredWriter *meteredWriter

	closeOnce            sync.Once
	closeHandlers        []SocketCloseHandler
	closeHandlersMutex   sync.RWMutex
	recycleHandlers      []func()
	recycleHandlersMutex sync.RWMutex
	recyclable           uint32

	prev *Socket
	next *Socket
}

// SocketHandler represents a signature of function used by Server to handle new connections.
type SocketHandler func(*Socket)

// SocketCloseHandler represents a signature of function used by Socket to register custom close handlers.
type SocketCloseHandler func(CloseReason)

// Close closes underlying TCP connection and executes all the registered close handlers.
func (s *Socket) Close(reason ...CloseReason) (err error) {
	s.closeOnce.Do(func() {
		if e := s.conn.Close(); e != nil {
			err = e
		}

		r := CloseReasonServer
		if reason != nil {
			r = reason[0]
		}

		s.closeHandlersMutex.RLock()
		for i := len(s.closeHandlers) - 1; i >= 0; i-- {
			handler := s.closeHandlers[i]
			handler(r)
		}
		s.closeHandlersMutex.RUnlock()
	})

	return
}

// Read conforms to the io.Reader interface.
func (s *Socket) Read(b []byte) (int, error) {
	n, err := s.reader.Read(b)
	if err != nil {
		if isBrokenPipe(err) {
			_ = s.Close(CloseReasonClient)
			return n, io.EOF
		}

		return n, err
	}

	return n, nil
}

// Write conforms to the io.Writer interface.
func (s *Socket) Write(b []byte) (int, error) {
	n, err := s.writer.Write(b)
	if err != nil {
		if isBrokenPipe(err) {
			_ = s.Close(CloseReasonClient)
			return n, io.EOF
		}

		return n, err
	}

	return n, nil
}

// SetDeadline sets deadline for underlying socket.
func (s *Socket) SetDeadline(deadline time.Time) error {
	err := s.conn.SetDeadline(deadline)
	if err != nil {
		if isBrokenPipe(err) {
			_ = s.Close(CloseReasonClient)
			return io.EOF
		}

		return err
	}

	return nil
}

// SetReadDeadline sets read deadline for underlying socket.
func (s *Socket) SetReadDeadline(deadline time.Time) error {
	err := s.conn.SetReadDeadline(deadline)
	if err != nil {
		if isBrokenPipe(err) {
			_ = s.Close(CloseReasonClient)
			return io.EOF
		}

		return err
	}

	return nil
}

// SetWriteDeadline sets read deadline for underlying socket.
func (s *Socket) SetWriteDeadline(deadline time.Time) error {
	err := s.conn.SetWriteDeadline(deadline)
	if err != nil {
		if isBrokenPipe(err) {
			_ = s.Close(CloseReasonClient)
			return io.EOF
		}

		return err
	}

	return nil
}

// RemoteAddress returns a remote address of the socket.
func (s *Socket) RemoteAddress() string {
	return s.remoteAddr
}

// ConnectedAt returns a unix timestamp indicating the exact moment the socket has connected (UTC, in milliseconds).
func (s *Socket) ConnectedAt() int64 {
	return s.timestamp
}

// OnClose registers a handler that is called when underlying TCP connection is being closed.
func (s *Socket) OnClose(handler SocketCloseHandler) {
	s.closeHandlersMutex.Lock()
	defer s.closeHandlersMutex.Unlock()

	s.closeHandlers = append(s.closeHandlers, handler)
}

// OnRecycle registers a handler that is called when the Socket object is being recycled and put back into pool.
func (s *Socket) OnRecycle(handler func()) {
	s.recycleHandlersMutex.Lock()
	defer s.recycleHandlersMutex.Unlock()

	s.recycleHandlers = append(s.recycleHandlers, handler)
}

// Unwrap returns underlying net.Conn instance from Socket.
func (s *Socket) Unwrap() net.Conn {
	return s.conn
}

// UnwrapTLS tries to return underlying tls.Conn instance from Socket.
func (s *Socket) UnwrapTLS() (*tls.Conn, bool) {
	if conn, ok := s.conn.(*tls.Conn); ok {
		return conn, true
	}

	return nil, false
}

// WrapReader allows to wrap reader object into user defined wrapper.
func (s *Socket) WrapReader(wrapper func(io.Reader) io.Reader) {
	s.reader = wrapper(s.reader)
}

// WrapWriter allows to wrap writer object into user defined wrapper.
func (s *Socket) WrapWriter(wrapper func(io.Writer) io.Writer) {
	s.writer = wrapper(s.writer)
}

// TotalRead returns a total number of bytes read through this socket.
func (s *Socket) TotalRead() uint64 {
	return s.meteredReader.Total()
}

// ReadLastSecond returns a total number of bytes read from this socket last second.
func (s *Socket) ReadLastSecond() uint64 {
	return s.meteredReader.PerSecond()
}

// TotalWritten returns a total number of bytes written through this socket.
func (s *Socket) TotalWritten() uint64 {
	return s.meteredWriter.Total()
}

// WrittenLastSecond returns a total number of bytes written to this socket last second.
func (s *Socket) WrittenLastSecond() uint64 {
	return s.meteredWriter.PerSecond()
}

func (s *Socket) init(conn net.Conn) {
	s.remoteAddr = parseRemoteAddress(conn)
	s.timestamp = time.Now().UTC().UnixMilli()
	s.conn = conn
	s.meteredReader.reader = conn
	s.meteredWriter.writer = conn
	s.reader = s.meteredReader
	s.writer = s.meteredWriter
}

func (s *Socket) reset() {
	s.remoteAddr = ""
	s.conn = nil
	s.meteredReader.reset()
	s.meteredWriter.reset()
	s.recyclable = 0
	s.closeHandlers = nil
	s.recycleHandlers = nil
	s.closeOnce = sync.Once{}
	s.closeHandlersMutex = sync.RWMutex{}
	s.recycleHandlersMutex = sync.RWMutex{}

	s.prev = nil
	s.next = nil
}

func (s *Socket) recycle() {
	s.recycleHandlersMutex.RLock()
	for i := len(s.recycleHandlers) - 1; i >= 0; i-- {
		handler := s.recycleHandlers[i]
		handler()
	}
	s.recycleHandlersMutex.RUnlock()

	atomic.StoreUint32(&s.recyclable, 1)
}

func (s *Socket) isRecyclable() bool {
	return atomic.LoadUint32(&s.recyclable) == 1
}

func (s *Socket) updateMetrics(interval time.Duration) (uint64, uint64) {
	reads := s.meteredReader.Update(interval)
	writes := s.meteredWriter.Update(interval)
	return reads, writes
}
