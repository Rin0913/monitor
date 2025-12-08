package worker

import (
	"context"
	"sync"
	"testing"
	"time"
)

type recordingWorker struct {
	id    int
	runFn func(ctx context.Context) error
}

func (w *recordingWorker) Run(ctx context.Context) error {
	return w.runFn(ctx)
}

func TestManager_StartAndStop_ThreeWorkers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	runCounts := make(map[int]int)
	started := make(chan int, 3)

	factory := func(id int) Worker {
		return &recordingWorker{
			id: id,
			runFn: func(ctx context.Context) error {
				mu.Lock()
				runCounts[id]++
				mu.Unlock()
				started <- id
				<-ctx.Done()
				return ctx.Err()
			},
		}
	}

	mgr := NewManager(3, 50*time.Millisecond, factory)
	mgr.Start(ctx)

	waitAllStarted := time.After(2 * time.Second)
loop:
	for {
		mu.Lock()
		if len(runCounts) == 3 {
			mu.Unlock()
			break loop
		}
		mu.Unlock()

		select {
		case <-waitAllStarted:
			t.Fatalf("workers did not all start in time, got=%v", runCounts)
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancel()
	mgr.Stop()

	mu.Lock()
	defer mu.Unlock()

	if len(runCounts) != 3 {
		t.Fatalf("expected 3 workers to have run, got=%d", len(runCounts))
	}
	for id, c := range runCounts {
		if c != 1 {
			t.Fatalf("worker %d run count = %d, expected 1", id, c)
		}
	}
}
