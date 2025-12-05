package worker

import (
	"context"
	"errors"
	"time"
    "log"

	"github.com/Rin0913/monitor/internal/health"
	"github.com/Rin0913/monitor/internal/scheduler"
)

type InternalWorker struct {
	engine     *Engine
	scheduler  *scheduler.Scheduler
	healthRepo health.Repository
}

func NewInternalWorker(engine *Engine, repo health.Repository, s *scheduler.Scheduler) *InternalWorker {
	return &InternalWorker{
		engine:     engine,
		scheduler:  s,
		healthRepo: repo,
	}
}

func (w *InternalWorker) Run(ctx context.Context) error {
	log.Println("[INFO] internal worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("[INFO] internal worker stopped: context canceled")
			return ctx.Err()
		default:
		}

		job, err := w.scheduler.NextJob(ctx)
		if err != nil {
			if errors.Is(err, scheduler.ErrClosed) {
				log.Println("[INFO] scheduler closed, worker exiting")
				return err
			}
			if errors.Is(err, ctx.Err()) {
				log.Println("[INFO] worker exiting due to context cancellation")
				return err
			}

			log.Printf("[ERROR] scheduler.NextJob failed: %v\n", err)
			continue
		}

		if job == nil {
			log.Println("[DEBUG] no job available, sleeping 1s")
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("[INFO] received job: deviceID=%s method=%s address=%s\n",
			job.DeviceID, job.Method, job.Address)

		h := w.engine.Handle(ctx, job)
		if h == nil {
			log.Printf("[WARN] handler returned nil health status for deviceID=%s\n", job.DeviceID)
			continue
		}

		ttl := time.Duration(job.TimeoutS*3) * time.Second
		if err := w.healthRepo.Save(ctx, h, ttl); err != nil {
			log.Printf("[ERROR] failed to save health status for deviceID=%s: %v\n",
				job.DeviceID, err)
			continue
		}

		log.Printf("[INFO] health updated: deviceID=%s status=%s latency=%dms\n",
			h.DeviceID, h.Status, h.Latency)
	}
}

