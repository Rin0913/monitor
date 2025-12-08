package worker

import (
	"context"
	"log"
	"time"
)

type RemoteWorker struct {
	name      string
	engine    *Engine
	serverURL string
	workerID  string
	key       string
}

func NewRemoteWorker(name string, engine *Engine, serverURL, workerID, key string) *RemoteWorker {
	return &RemoteWorker{
		name:      name,
		engine:    engine,
		serverURL: serverURL,
		workerID:  workerID,
		key:       key,
	}
}

func (w *RemoteWorker) Run(ctx context.Context) error {
	log.Printf("[INFO] remote worker %s started\n", w.name)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] remote worker %s stopped: context canceled\n", w.name)
			return ctx.Err()
		default:
		}

		job, status, err := PollJob(w.serverURL, w.workerID, w.key)
		if err != nil {
			log.Printf("[ERROR] %s poll job failed: %v\n", w.name, err)
			time.Sleep(1 * time.Second)
			continue
		}

		if status == 204 || job == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		if status != 200 {
			log.Printf("[WARN] worker %s poll returned status %d\n", w.name, status)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("[INFO] remote worker %s received job: deviceID=%s method=%s address=%s\n",
			w.name, job.DeviceID, job.Method, job.Address)

		h := w.engine.Handle(ctx, job)
		if h == nil {
			log.Printf("[WARN] remote worker %s handler returned nil health status for deviceID=%s\n",
				w.name, job.DeviceID)
			continue
		}

		if h.Runner == "" {
			h.Runner = w.workerID
		}

		code, err := ReportJob(w.serverURL, w.key, h)
		if err != nil {
			log.Printf("[ERROR] remote worker %s report failed for deviceID=%s: %v\n",
				w.name, h.DeviceID, err)
			continue
		}

		if code >= 300 {
			log.Printf("[WARN] remote worker %s report returned status %d for deviceID=%s\n",
				w.name, code, h.DeviceID)
			continue
		}

		log.Printf("[INFO] remote worker %s health reported: deviceID=%s status=%s latency=%dms\n",
			w.name, h.DeviceID, h.Status, h.Latency)
	}
}
