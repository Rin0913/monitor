package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Rin0913/monitor/internal/app/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	workerNum := 1
	if n, err := strconv.Atoi(os.Getenv("LOCAL_WORKER_NUM")); err == nil && n >= 0 {
		workerNum = n
	}

	if err := server.Run(ctx, workerNum); err != nil {
		log.Fatalf("[ERROR] server exited with error: %v", err)
	} else {
		log.Println("[INFO] http server stopped")
	}

	log.Println("[INFO] Goodbye!")
}
