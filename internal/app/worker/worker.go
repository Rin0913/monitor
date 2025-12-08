package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Rin0913/monitor/internal/worker"
)

func Run(ctx context.Context, serverURL string, workerID string, workerKey string, workerNum int) error {
	engine := worker.NewEngine()
	_ = engine.LoadConfig("checkers.yaml")

	manager := worker.NewManager(workerNum, 2*time.Second, func(id int) worker.Worker {
		return worker.NewRemoteWorker(
			fmt.Sprintf("%s#%d", workerID, id),
			engine,
			serverURL,
			workerID,
			workerKey,
		)
	})

	manager.Start(ctx)
	defer manager.Stop()

	<-ctx.Done()
	log.Println("[INFO] shutdown signal received")
	return nil
}
