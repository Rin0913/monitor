package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	servercmd "github.com/Rin0913/monitor/internal/app/server"
	workercmd "github.com/Rin0913/monitor/internal/app/worker"
)

func startServer(t *testing.T, workerNum int) (context.Context, context.CancelFunc, chan error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- servercmd.Run(ctx, workerNum)
	}()

	return ctx, cancel, errCh
}

func startWorker(t *testing.T, serverURL string, workerKey string) (context.Context, context.CancelFunc, chan error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- workercmd.Run(ctx, serverURL, "test-worker", workerKey, 1)
	}()

	return ctx, cancel, errCh
}

func waitForShutdown(t *testing.T, errCh chan error) {
	t.Helper()

	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("run(ctx) did not exit after cancel")
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run(ctx) returned error: %v", err)
		}
	}
}

func waitForHealthReady(t *testing.T, client *http.Client, baseURL string) *http.Response {
	t.Helper()

	var resp *http.Response
	var err error

	for i := 0; i < 10; i++ {
		resp, err = client.Get(baseURL + "/health")
		if err == nil {
			return resp
		}
		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("failed to call /health: %v", err)
	return nil
}
