package httpserver

import (
    "os"
    "context"
	"net/http"

	"github.com/redis/go-redis/v9"
	"github.com/Rin0913/monitor/internal/device"
	"github.com/Rin0913/monitor/internal/health"
    "github.com/Rin0913/monitor/internal/scheduler"
)

type Server struct {
	deviceRepo device.Repository
	healthRepo health.Repository
    scheduler *scheduler.Scheduler

	presharedWorkerKey string
}

func NewServer(redisClient *redis.Client) *Server {
    deviceRepo := device.NewRedisRepository(redisClient)
    healthRepo := health.NewRedisRepository(redisClient)
    scheduler := scheduler.New(deviceRepo, healthRepo)

    scheduler.Bootstrap(context.Background())

	return &Server{
		deviceRepo: deviceRepo,
        healthRepo: healthRepo,
        scheduler: scheduler,
        presharedWorkerKey: os.Getenv("PRESHARED_WORKER_KEY"),
	}
}

func (s *Server) Scheduler() *scheduler.Scheduler {
    return s.scheduler
}

func (s *Server) HealthRepo() health.Repository {
    return s.healthRepo
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	s.registerHealthRoutes(mux)
	s.registerDeviceRoutes(mux)
	s.registerInternalRoutes(mux)
}

