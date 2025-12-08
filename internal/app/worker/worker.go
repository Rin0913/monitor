package worker

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Rin0913/monitor/internal/worker"
)

func Run(ctx context.Context, serverURL string, workerID string, workerKey string, workerNum int) error {
	engine := worker.NewEngine()
	_ = engine.LoadConfig("checkers.yaml")

	for i := 0; i < workerNum; i++ {
		w := worker.NewRemoteWorker(
			fmt.Sprintf("%s#%d", workerID, i+1),
			engine,
			serverURL,
			workerID,
			workerKey,
		)

		go func(id int) {
			if err := w.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("[WARN] internal worker %d stopped with error: %v", id, err)
			} else {
				log.Printf("[INFO] internal worker %d stopped", id)
			}
		}(i + 1)
	}

	<-ctx.Done()
	log.Println("[INFO] shutdown signal received")
	return nil
}
