package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Rin0913/monitor/internal/httpserver"
	"github.com/Rin0913/monitor/internal/redisclient"
	"github.com/Rin0913/monitor/internal/worker"
)

func Run(ctx context.Context, workerNum int) error {
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

	manager := worker.NewManager(workerNum, 2*time.Second, func(id int) worker.Worker {
		return worker.NewInternalWorker(
			fmt.Sprintf("internal#%d", id),
			engine,
			httpServer.HealthRepo(),
			httpServer.Scheduler(),
		)
	})

	manager.Start(ctx)
	defer manager.Stop()

	errCh := make(chan error, 1)

	go func() {
		log.Printf("[INFO] http server listening on %s", addr)
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("[INFO] shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil

	case err := <-errCh:
		return err
	}
}
