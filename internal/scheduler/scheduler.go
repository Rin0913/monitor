package scheduler

import (
    "log"
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Rin0913/monitor/internal/device"
	"github.com/Rin0913/monitor/internal/health"
)

var ErrClosed = errors.New("scheduler closed")

type CheckJob struct {
	DeviceID    string
	Address     string
	Method      string
	IntervalSec int
	TimeoutS    int
	nextRun     time.Time
	index       int
}

type Scheduler struct {
	mu         sync.Mutex
	cond       *sync.Cond
	jobs       jobHeap
	closed     bool
	deviceRepo device.Repository
	healthRepo health.Repository
}

func New(deviceRepo device.Repository, healthRepo health.Repository) *Scheduler {
	s := &Scheduler{
		deviceRepo: deviceRepo,
		healthRepo: healthRepo,
	}
	s.cond = sync.NewCond(&s.mu)
	heap.Init(&s.jobs)
	return s
}

func (s *Scheduler) Bootstrap(ctx context.Context) error {
	devices, err := s.deviceRepo.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()

	for _, d := range devices {
		h, err := s.healthRepo.Get(ctx, d.ID)

        if(err != nil) {
            log.Printf("[WARN] %v", err)
        }

		interval := time.Duration(d.IntervalSec) * time.Second
		if interval <= 0 {
			interval = 60 * time.Second
		}

		var nextRun time.Time

		if h == nil || h.LastCheck.IsZero() {
			nextRun = now
		} else {
			scheduled := h.LastCheck.Add(interval)
			if scheduled.Before(now) {
				nextRun = now
			} else {
				nextRun = scheduled
			}
		}

		s.addWithNextRun(d, nextRun)
	}

	return nil
}

func (s *Scheduler) Add(d *device.Device) {
	s.addWithNextRun(d, time.Now())
}


func (s *Scheduler) addWithNextRun(d *device.Device, t time.Time) {
	job := &CheckJob{
		DeviceID:    d.ID,
		Address:     d.Address,
		Method:      d.CheckMethod,
		IntervalSec: d.IntervalSec,
		TimeoutS:    d.IntervalSec,
		nextRun:     t,
	}
	s.add(job)
}

func (s *Scheduler) add(job *CheckJob) {
	s.mu.Lock()
	heap.Push(&s.jobs, job)
	s.mu.Unlock()
	s.cond.Signal()
}

func (s *Scheduler) NextJob(ctx context.Context) (*CheckJob, error) {
	for {
		s.mu.Lock()

		if ctx.Err() != nil {
			s.mu.Unlock()
			return nil, ctx.Err()
		}
		if s.closed {
			s.mu.Unlock()
			return nil, ErrClosed
		}

		if len(s.jobs) == 0 {
			s.cond.Wait()
			s.mu.Unlock()
			continue
		}

		top := s.jobs[0]
		now := time.Now()
		if top.nextRun.After(now) {
			wait := top.nextRun.Sub(now)
			s.mu.Unlock()

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
			continue
		}

		job := heap.Pop(&s.jobs).(*CheckJob)

		interval := time.Duration(job.IntervalSec) * time.Second
		if interval <= 0 {
			interval = 60 * time.Second
		}
		next := job.nextRun.Add(interval)
		if next.Before(now) {
			next = now.Add(interval)
		}

		nextJob := *job
		nextJob.nextRun = next
		heap.Push(&s.jobs, &nextJob)

		s.mu.Unlock()
		return job, nil
	}
}

