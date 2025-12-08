package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Rin0913/monitor/internal/app/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverURL := os.Getenv("MONITOR_SERVER_URL")
	workerID := os.Getenv("WORKER_ID")
	workerKey := os.Getenv("WORKER_KEY")

	if serverURL == "" || workerID == "" {
		log.Fatalf("missing MONITOR_SERVER_URL or WORKER_ID")
	}

	workerNum := 1
	if n, err := strconv.Atoi(os.Getenv("WORKER_NUM")); err == nil && n > 0 {
		workerNum = n
	}

	if err := worker.Run(ctx, serverURL, workerID, workerKey, workerNum); err != nil {
		log.Fatalf("[ERROR] worker exited with error: %v", err)
	} else {
		log.Println("[INFO] worker stopped")
	}

	log.Println("[INFO] See you!")
}
