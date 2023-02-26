package tinytcp

import (
	"fmt"
	"sync"
	"time"
)

type backgroundJob struct {
	fn           func()
	panicHandler func(error)
	interval     time.Duration

	ticker  *time.Ticker
	m       sync.Mutex
	running bool
}

func newBackgroundJob(interval time.Duration, fn func(), panicHandler func(error)) *backgroundJob {
	return &backgroundJob{
		fn:           fn,
		panicHandler: panicHandler,
		interval:     interval,
	}
}

func (b *backgroundJob) Start() {
	b.m.Lock()
	defer b.m.Unlock()

	if b.running {
		return
	}
	b.running = true

	go func() {
		defer func() {
			if r := recover(); r != nil {
				b.panicHandler(fmt.Errorf("%v", r))
			}
		}()

		b.ticker = time.NewTicker(b.interval)

		for range b.ticker.C {
			b.m.Lock()

			if !b.running {
				break
			}

			b.fn()

			b.m.Unlock()
		}
	}()
}

func (b *backgroundJob) Stop() {
	b.m.Lock()
	defer b.m.Unlock()

	if !b.running {
		return
	}
	b.running = false

	b.ticker.Stop()
}
