package worker

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/Rin0913/monitor/internal/health"
	"github.com/Rin0913/monitor/internal/scheduler"
)

type CheckerFunc func(ctx context.Context, job *scheduler.CheckJob) (string, int, map[string]interface{}, error)

type Engine struct {
	mu       sync.RWMutex
	checkers map[string]CheckerFunc
}

func NewEngine() *Engine {
	e := &Engine{
		checkers: make(map[string]CheckerFunc),
	}
	e.RegisterChecker("tcp_check", tcpChecker)
	return e
}

func (e *Engine) RegisterChecker(method string, fn CheckerFunc) {
	if method == "" || fn == nil {
		return
	}
	e.mu.Lock()
	e.checkers[method] = fn
	e.mu.Unlock()
}

func (e *Engine) getChecker(method string) CheckerFunc {
	e.mu.RLock()
	fn := e.checkers[method]
	e.mu.RUnlock()
	return fn
}

func (e *Engine) Handle(ctx context.Context, job *scheduler.CheckJob) *health.HealthStatus {
	if job == nil || job.DeviceID == "" {
		return nil
	}

	fn := e.getChecker(job.Method)
	if fn == nil {
		return &health.HealthStatus{
			DeviceID:  job.DeviceID,
			Status:    "UNKNOWN_METHOD",
			Latency:   -1,
			LastCheck: time.Now(),
			Data: map[string]interface{}{
				"method": job.Method,
			},
		}
	}

	timeout := time.Duration(job.TimeoutS) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	jobCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	status, latency, data, err := fn(jobCtx, job)
	if status == "" {
		if err != nil {
			status = "DOWN"
		} else {
			status = "UNKNOWN"
		}
	}
	if data == nil {
		data = make(map[string]interface{})
	}
	if err != nil {
		data["error"] = err.Error()
	}

	return &health.HealthStatus{
		DeviceID:  job.DeviceID,
		Status:    status,
		Latency:   latency,
		LastCheck: time.Now(),
		Data:      data,
	}
}

func tcpChecker(ctx context.Context, job *scheduler.CheckJob) (string, int, map[string]interface{}, error) {
	start := time.Now()

	timeout := time.Duration(job.TimeoutS) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	dialer := net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", job.Address)
	latency := int(time.Since(start) / time.Millisecond)
	if err != nil {
		return "DOWN", latency, nil, err
	}
	_ = conn.Close()

	return "UP", latency, nil, nil
}
