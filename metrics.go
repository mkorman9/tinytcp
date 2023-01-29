package tinytcp

import (
	"io"
	"sync/atomic"
	"time"
)

// ServerMetrics contains metrics collected from TCP server.
type ServerMetrics struct {
	// TotalRead is total number of bytes read by the server.
	TotalRead uint64

	// TotalWritten is total number of bytes written by the server.
	TotalWritten uint64

	// ReadLastSecond is total number of bytes read by the server last second.
	ReadLastSecond uint64

	// WrittenLastSecond is total number of bytes written by the server last second.
	WrittenLastSecond uint64

	// Connections is a total number of active connections during the last second.
	Connections int

	// Goroutines is a total number of active goroutines during the last second.
	Goroutines int
}

type meteredReader struct {
	reader  io.Reader
	total   uint64
	current uint64
	rate    uint64
}

func (r *meteredReader) Read(b []byte) (int, error) {
	n, err := r.reader.Read(b)

	if n > 0 {
		atomic.AddUint64(&r.current, uint64(n))
	}

	return n, err
}

func (r *meteredReader) Total() uint64 {
	return atomic.LoadUint64(&r.total)
}

func (r *meteredReader) PerSecond() uint64 {
	return atomic.LoadUint64(&r.rate)
}

func (r *meteredReader) Update(interval time.Duration) uint64 {
	current := atomic.SwapUint64(&r.current, 0)

	atomic.StoreUint64(&r.rate, uint64(float64(current)/interval.Seconds()))
	atomic.AddUint64(&r.total, current)

	return current
}

func (r *meteredReader) reset() {
	r.reader = nil
	r.total = 0
	r.current = 0
	r.rate = 0
}

type meteredWriter struct {
	writer  io.Writer
	total   uint64
	current uint64
	rate    uint64
}

func (w *meteredWriter) Write(b []byte) (int, error) {
	n, err := w.writer.Write(b)

	if n > 0 {
		atomic.AddUint64(&w.current, uint64(n))
	}

	return n, err
}

func (w *meteredWriter) Total() uint64 {
	return atomic.LoadUint64(&w.total)
}

func (w *meteredWriter) PerSecond() uint64 {
	return atomic.LoadUint64(&w.rate)
}

func (w *meteredWriter) Update(interval time.Duration) uint64 {
	current := atomic.SwapUint64(&w.current, 0)

	atomic.StoreUint64(&w.rate, uint64(float64(current)/interval.Seconds()))
	atomic.AddUint64(&w.total, current)

	return current
}

func (w *meteredWriter) reset() {
	w.writer = nil
	w.total = 0
	w.current = 0
	w.rate = 0
}
