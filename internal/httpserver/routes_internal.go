package httpserver

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Rin0913/monitor/internal/health"
	"github.com/Rin0913/monitor/internal/scheduler"
)

type workerPollRequest struct {
	WorkerID string `json:"worker_id"`
}

type workerReportRequest struct {
	WorkerID  string     `json:"worker_id"`
	JobID     string     `json:"job_id"`
	DeviceID  string     `json:"device_id"`
	Status    string     `json:"status"`
	LatencyMS int        `json:"latency_ms"`
	LastCheck *time.Time `json:"last_check,omitempty"`
}

func (s *Server) verifyWorkerRequest(r *http.Request, body []byte) bool {
	if s.presharedWorkerKey == "" {
		return true
	}

	id := r.Header.Get("X-Worker-Id")
	tsStr := r.Header.Get("X-Worker-Timestamp")
	sig := r.Header.Get("X-Worker-Signature")
	if id == "" || tsStr == "" || sig == "" {
		return false
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return false
	}

	now := time.Now().Unix()
	if ts > now+300 || ts < now-300 {
		return false
	}

	var buf bytes.Buffer
	buf.WriteString(tsStr)
	buf.WriteByte('\n')
	buf.WriteString(id)
	buf.WriteByte('\n')
	buf.WriteString(r.Method)
	buf.WriteByte('\n')
	buf.WriteString(r.URL.Path)
	buf.WriteByte('\n')
	buf.Write(body)

	mac := hmac.New(sha256.New, []byte(s.presharedWorkerKey))
	mac.Write(buf.Bytes())
	expected := mac.Sum(nil)

	got, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}

	if !hmac.Equal(got, expected) {
		return false
	}
	return true
}

func (s *Server) workerPollJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))

	if !s.verifyWorkerRequest(r, body) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req workerPollRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	job, err := s.scheduler.TryNextJob(r.Context())
	if err != nil {
		if errors.Is(err, scheduler.ErrClosed) {
			http.Error(w, "scheduler closed", http.StatusServiceUnavailable)
			return
		}
		if errors.Is(err, r.Context().Err()) {
			return
		}
		http.Error(w, "failed to get job", http.StatusInternalServerError)
		return
	}
	if job == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(job)
}

func (s *Server) workerReportJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))

	if !s.verifyWorkerRequest(r, body) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req workerReportRequest
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.DeviceID == "" {
		http.Error(w, "missing device_id", http.StatusBadRequest)
		return
	}

	checkedAt := time.Now()
	if req.LastCheck != nil && !req.LastCheck.IsZero() {
		checkedAt = *req.LastCheck
	}

	h := &health.HealthStatus{
		DeviceID:  req.DeviceID,
		Status:    req.Status,
		Latency:   req.LatencyMS,
		Runner:    req.WorkerID,
		LastCheck: checkedAt,
		Data: map[string]interface{}{
			"job_id": req.JobID,
		},
	}

	if err := s.healthRepo.Save(r.Context(), h, 5*time.Minute); err != nil {
		http.Error(w, "failed to save health", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) registerInternalRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/worker/jobs/poll", s.workerPollJob)
	mux.HandleFunc("POST /internal/worker/jobs/report", s.workerReportJob)
}
