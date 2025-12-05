package main

import (
    "context"
	"log"
	"net/http"
	"time"

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

    iw := worker.NewInternalWorker(
        worker.NewEngine(),
        httpServer.HealthRepo(),
        httpServer.Scheduler())

    ctx := context.Background()

    go func() {
        _ = iw.Run(ctx)
    }()

    log.Printf("listening on %s", addr)
    log.Fatal(s.ListenAndServe())
}
