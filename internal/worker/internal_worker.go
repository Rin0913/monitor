package worker

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Rin0913/monitor/internal/health"
	"github.com/Rin0913/monitor/internal/scheduler"
)

type InternalWorker struct {
	name       string
	engine     *Engine
	scheduler  *scheduler.Scheduler
	healthRepo health.Repository
}

func NewInternalWorker(name string, engine *Engine, repo health.Repository, s *scheduler.Scheduler) *InternalWorker {
	return &InternalWorker{
		name:       name,
		engine:     engine,
		scheduler:  s,
		healthRepo: repo,
	}
}

func (w *InternalWorker) Run(ctx context.Context) error {
	log.Printf("[INFO] internal worker %s started\n", w.name)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] internal worker %s stopped: context canceled\n", w.name)
			return ctx.Err()
		default:
		}

		job, err := w.scheduler.NextJob(ctx)
		if err != nil {
			if errors.Is(err, scheduler.ErrClosed) {
				log.Printf("[INFO] scheduler closed, worker %s exiting", w.name)
				return err
			}
			if errors.Is(err, ctx.Err()) {
				log.Printf("[INFO] worker %s exiting due to context cancellation", w.name)
				return err
			}

			log.Printf("[ERROR] %s scheduler.NextJob failed: %v\n", w.name, err)
			continue
		}

		if job == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("[INFO] worker %s received job: deviceID=%s method=%s address=%s\n",
			w.name, job.DeviceID, job.Method, job.Address)

		h := w.engine.Handle(ctx, job)
		if h == nil {
			log.Printf("[WARN] worker %s handler returned nil health status for deviceID=%s\n",
				w.name, job.DeviceID)
			continue
		}

		ttl := time.Duration(job.TimeoutS*3) * time.Second

		h.Runner = w.name

		if err := w.healthRepo.Save(ctx, h, ttl); err != nil {
			log.Printf("[ERROR] worker %s failed to save health status for deviceID=%s: %v\n",
				w.name, job.DeviceID, err)
			continue
		}

		log.Printf("[INFO] worker %s health updated: deviceID=%s status=%s latency=%dms\n",
			w.name, h.DeviceID, h.Status, h.Latency)
	}
}
