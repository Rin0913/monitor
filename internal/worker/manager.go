package worker

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

type Worker interface {
	Run(ctx context.Context) error
}

type Factory func(id int) Worker

type Manager struct {
	factory     Factory
	num         int
	backoff     time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	startedOnce sync.Once
}

func NewManager(num int, backoff time.Duration, f Factory) *Manager {
	return &Manager{
		factory: f,
		num:     num,
		backoff: backoff,
	}
}

func (m *Manager) Start(parent context.Context) {
	m.startedOnce.Do(func() {
		m.ctx, m.cancel = context.WithCancel(parent)
		for i := 0; i < m.num; i++ {
			id := i + 1
			m.wg.Add(1)
			go m.runOne(id)
		}
	})
}

func (m *Manager) runOne(id int) {
	defer m.wg.Done()

	for {
		w := m.factory(id)
		err := w.Run(m.ctx)

		if err == nil || errors.Is(err, context.Canceled) {
			return
		}

		log.Printf("[WARN] worker %d stopped with error: %v", id, err)

		select {
		case <-time.After(m.backoff):
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
}
