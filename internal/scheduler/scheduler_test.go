package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Rin0913/monitor/internal/device"
	"github.com/Rin0913/monitor/internal/health"
)

func TestNextJobReschedule(t *testing.T) {
	s := New(nil, nil)

	now := time.Now()
	job := &CheckJob{
		DeviceID:    "dev1",
		Address:     "1.2.3.4:80",
		Method:      "tcp",
		IntervalSec: 1,
		TimeoutS:    1,
		nextRun:     now,
	}
	s.add(job)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	j1, err := s.NextJob(ctx)
	if err != nil {
		t.Fatalf("NextJob first: %v", err)
	}
	if j1.DeviceID != "dev1" {
		t.Fatalf("unexpected deviceID: %s", j1.DeviceID)
	}

	j2, err := s.NextJob(ctx)
	if err != nil {
		t.Fatalf("NextJob second: %v", err)
	}
	if j2.DeviceID != "dev1" {
		t.Fatalf("unexpected deviceID second: %s", j2.DeviceID)
	}
	if !j2.nextRun.After(j1.nextRun) {
		t.Fatalf("nextRun not advanced: first=%v second=%v", j1.nextRun, j2.nextRun)
	}
}

func TestBootstrapUsesHealthForNextRun(t *testing.T) {
	now := time.Now()

	d1 := &device.Device{
		ID:          "no-health",
		Address:     "1.1.1.1:80",
		CheckMethod: "tcp",
		IntervalSec: 60,
	}
	d2 := &device.Device{
		ID:          "fresh",
		Address:     "8.8.8.8:80",
		CheckMethod: "tcp",
		IntervalSec: 60,
	}

	devRepo := &fakeDeviceRepo{devs: []*device.Device{d1, d2}}

	healthRepo := &fakeHealthRepo{
		m: map[string]*health.HealthStatus{
			"fresh": {
				LastCheck: now.Add(-30 * time.Second),
			},
		},
	}

	s := &Scheduler{
		deviceRepo: devRepo,
		healthRepo: healthRepo,
	}
	s.cond = sync.NewCond(&s.mu)
	heap.Init(&s.jobs)

	ctx := context.Background()
	if err := s.Bootstrap(ctx); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	j1, err := s.NextJob(ctx)
	if err != nil {
		t.Fatalf("NextJob: %v", err)
	}
	if j1.DeviceID != "no-health" {
		t.Fatalf("first job should be no-health, got %s", j1.DeviceID)
	}

	j2, err := s.TryNextJob(ctx)
	if j2 != nil {
		t.Fatalf("second job is not produced yet")
	}
	if err != nil {
		t.Fatalf("error occurs in unblocking TryNextJob: %v", err)
	}
}

// Some trivial definitions

type fakeDeviceRepo struct {
	devs []*device.Device
}

func (r *fakeDeviceRepo) List(ctx context.Context) ([]*device.Device, error) {
	return r.devs, nil
}

func (r *fakeDeviceRepo) Save(ctx context.Context, d *device.Device) error {
	return nil
}

func (r *fakeDeviceRepo) GetByID(ctx context.Context, id string) (*device.Device, error) {
	return nil, nil
}
func (r *fakeDeviceRepo) DeleteByID(ctx context.Context, id string) error {
	return nil
}

type fakeHealthRepo struct {
	m map[string]*health.HealthStatus
}

func (r *fakeHealthRepo) Get(ctx context.Context, id string) (*health.HealthStatus, error) {
	if r.m == nil {
		return nil, nil
	}
	return r.m[id], nil
}

func (r *fakeHealthRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (r *fakeHealthRepo) Save(ctx context.Context, h *health.HealthStatus, ttl time.Duration) error {
	if r.m == nil {
		r.m = make(map[string]*health.HealthStatus)
	}
	r.m[h.DeviceID] = h
	return nil
}
