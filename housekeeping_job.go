package tinytcp

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type housekeepingJob struct {
	fn           func()
	panicHandler func(error)
	interval     time.Duration

	ticker  *time.Ticker
	m       sync.Mutex
	running bool
}

func newHousekeepingJob(interval time.Duration, fn func(), panicHandler func(error)) *housekeepingJob {
	return &housekeepingJob{
		fn:           fn,
		panicHandler: panicHandler,
		interval:     interval,
	}
}

func (h *housekeepingJob) Start() {
	h.m.Lock()
	defer h.m.Unlock()

	if h.running {
		return
	}
	h.running = true

	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.panicHandler(fmt.Errorf("%v", r))
			}
		}()

		h.ticker = time.NewTicker(h.interval)

		for range h.ticker.C {
			err := func() error {
				h.m.Lock()
				defer h.m.Unlock()

				if !h.running {
					return io.EOF
				}

				h.fn()
				return nil
			}()

			if err != nil {
				break
			}
		}
	}()
}

func (h *housekeepingJob) Stop() {
	h.m.Lock()
	defer h.m.Unlock()

	if !h.running {
		return
	}
	h.running = false

	h.ticker.Stop()
}
