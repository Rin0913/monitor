package main

import (
    "os"
    "fmt"
    "errors"
    "strconv"
    "context"
	"log"
	"net/http"
	"time"
    "os/signal"
    "syscall"

    "github.com/Rin0913/monitor/internal/httpserver"
    "github.com/Rin0913/monitor/internal/redisclient"
    "github.com/Rin0913/monitor/internal/worker"
)

func main() {
    redisClient := redisclient.NewClientFromEnv()
    httpServer := httpserver.NewServer(redisClient)
	
    mux := http.NewServeMux()

    httpServer.RegisterRoutes(mux)

    addr := ":8080"
	s := &http.Server{
		Addr:           addr,
		Handler:        mux,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

    engine := worker.NewEngine()
    _ = engine.LoadConfig("checkers.yaml")

    workerNum, _ := strconv.Atoi(os.Getenv("LOCAL_WORKER_NUM"))
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

    for i := 0; i < workerNum; i++ {
        iw := worker.NewInternalWorker(
            fmt.Sprintf("internal%d", i + 1),
            engine,
            httpServer.HealthRepo(),
            httpServer.Scheduler(),
        )

        go func(id int) {
            _ = iw.Run(ctx)
        }(i + 1)
    }

    go func() {
        log.Printf("[INFO] http server listening on %s", addr)
        err := s.ListenAndServe()
        if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[ERROR] http server error: %v", err)
		}
    }()

    <-ctx.Done()
    log.Println("[INFO] shutdown signal received")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

    if err := s.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] http server shutdown error: %v", err)
    } else {
        log.Println("[INFO] http server stopped")
    }

	log.Println("[INFO] Goodbye!")
}
