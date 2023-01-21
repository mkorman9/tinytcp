package tinytcp

import (
	"io"
	"sync"
	"sync/atomic"
)

// SocketRef allows to hold a reference to a socket outside its designated handler.
// tinytcp performs no memory allocations on critical paths, which means it needs to pool some objects.
// These pooled objects are typically structures that exist only in the context of a specific connection (eg. Sockets).
// Objects are returned to the pool as they are no longer needed and reused by other connection in the future.
// The rule is that a socket instance is only valid inside its designated handler and storing it outside this handler
// might result in some very nasty bugs. SocketRef provides a way to safely store a reference to a socket,
// and provide a subset of its functionalities.
type SocketRef struct {
	s        *Socket
	m        sync.RWMutex
	recycled uint32
}

// NewSocketRef creates an instance of SocketReference.
func NewSocketRef(s *Socket) *SocketRef {
	ref := &SocketRef{
		s: s,
	}

	s.onRecycle(ref.onRecycle)
	return ref
}

// Write writes data to a socket only if it hasn't been recycled yet.
func (r *SocketRef) Write(b []byte) (int, error) {
	r.m.RLock()
	defer r.m.RUnlock()

	if atomic.LoadUint32(&r.recycled) == 1 {
		return 0, io.EOF
	}

	return r.s.Write(b)
}

func (r *SocketRef) onRecycle() {
	r.m.Lock()
	defer r.m.Unlock()

	atomic.StoreUint32(&r.recycled, 1)
}
