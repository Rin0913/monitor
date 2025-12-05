package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func startTestServer(t *testing.T) (context.Context, context.CancelFunc, chan error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- run(ctx)
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

func TestRun_HealthEndpoint(t *testing.T) {
	_, cancel, errCh := startTestServer(t)
	defer cancel()

	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	baseURL := "http://127.0.0.1:8080"

	resp := waitForHealthReady(t, client, baseURL)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code from /health: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read /health body error: %v", err)
	}
	if len(body) == 0 {
		t.Fatalf("empty response body from /health")
	}

	cancel()
	waitForShutdown(t, errCh)
}

func TestRun_DevicesFlow(t *testing.T) {
	_, cancel, errCh := startTestServer(t)
	defer cancel()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	baseURL := "http://127.0.0.1:8080"

	resp := waitForHealthReady(t, client, baseURL)
	resp.Body.Close()

	postJSON := func(path, payload string) *http.Response {
		r, err := client.Post(baseURL+path, "application/json", strings.NewReader(payload))
		if err != nil {
			t.Fatalf("POST %s error: %v", path, err)
		}
		return r
	}

	r1 := postJSON("/devices", `{"address":"8.8.8.8","name":"dns test","check_method":"cmd_ping"}`)
	if r1.StatusCode != http.StatusCreated && r1.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(r1.Body)
		r1.Body.Close()
		t.Fatalf("unexpected status from POST /devices (dns): %d body=%s", r1.StatusCode, string(b))
	}
	r1.Body.Close()

	r2 := postJSON("/devices", `{"address":"sandb0x.tw:80","name":"web server test","check_method":"tcp_check"}`)
	if r2.StatusCode != http.StatusCreated && r2.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(r2.Body)
		r2.Body.Close()
		t.Fatalf("unexpected status from POST /devices (web): %d body=%s", r2.StatusCode, string(b))
	}
	r2.Body.Close()

	resp, err := client.Get(baseURL + "/devices")
	if err != nil {
		t.Fatalf("GET /devices error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status from GET /devices: %d body=%s", resp.StatusCode, string(b))
	}

	type deviceResp struct {
		ID      string `json:"id"`
		Address string `json:"address"`
		Name    string `json:"name"`
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read /devices body error: %v", err)
	}
	var devices []deviceResp
	if err := json.Unmarshal(b, &devices); err != nil {
		t.Fatalf("unmarshal /devices response error: %v body=%s", err, string(b))
	}

	if len(devices) == 0 {
		t.Fatalf("no devices returned from /devices")
	}

	var dnsID, webID string
	for _, d := range devices {
		if d.Address == "8.8.8.8" {
			dnsID = d.ID
		}
		if d.Address == "sandb0x.tw:80" {
			webID = d.ID
		}
	}

	if dnsID == "" || webID == "" {
		t.Fatalf("expected devices not found in /devices list, got=%v", devices)
	}

	type healthResp struct {
		Status    string `json:"status"`
		LatencyMS int    `json:"latency_ms"`
		LastCheck string `json:"last_check"`
	}

	getHealthOnce := func(id string) (healthResp, int) {
		url := baseURL + "/devices/" + id
		r, err := client.Get(url)
		if err != nil {
			return healthResp{}, 0
		}
		defer r.Body.Close()

		code := r.StatusCode
		data, err := io.ReadAll(r.Body)
		if err != nil {
			return healthResp{}, code
		}

		var h healthResp
		if err := json.Unmarshal(data, &h); err != nil {
			return healthResp{}, code
		}
		return h, code
	}

	waitHealthReady := func(id string) healthResp {
		var h healthResp
		for i := 0; i < 20; i++ {
			h, _ = getHealthOnce(id)
			if h.Status != "" && h.Status != "unknown" {
				return h
			}
			time.Sleep(2 * time.Second)
		}
		t.Fatalf("health for device %s stayed unknown or empty too long: %+v", id, h)
		return healthResp{}
	}

	hDNS := waitHealthReady(dnsID)
	hWeb := waitHealthReady(webID)

	if hDNS.Status == "" || hDNS.Status == "unknown" {
		t.Fatalf("unexpected dns health status: %+v", hDNS)
	}
	if hWeb.Status == "" || hWeb.Status == "unknown" {
		t.Fatalf("unexpected web health status: %+v", hWeb)
	}

	cancel()
	waitForShutdown(t, errCh)
}
