package worker

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Rin0913/monitor/internal/health"
	"github.com/Rin0913/monitor/internal/scheduler"
)

func sign(key, workerID, method, path string, body []byte, ts int64) (string, string, string) {
	tsStr := strconv.FormatInt(ts, 10)

	var buf bytes.Buffer
	buf.WriteString(tsStr)
	buf.WriteByte('\n')
	buf.WriteString(workerID)
	buf.WriteByte('\n')
	buf.WriteString(method)
	buf.WriteByte('\n')
	buf.WriteString(path)
	buf.WriteByte('\n')
	buf.Write(body)

	m := hmac.New(sha256.New, []byte(key))
	m.Write(buf.Bytes())
	s := hex.EncodeToString(m.Sum(nil))

	return workerID, tsStr, s
}

func PollJob(serverURL, workerID, key string) (*scheduler.CheckJob, int, error) {
	body, _ := json.Marshal(map[string]string{"worker_id": workerID})
	path := "/internal/worker/jobs/poll"
	ts := time.Now().Unix()

	id, tsStr, sig := sign(key, workerID, http.MethodPost, path, body, ts)

	req, _ := http.NewRequest(http.MethodPost, serverURL+path, bytes.NewReader(body))
	req.Header.Set("X-Worker-Id", id)
	req.Header.Set("X-Worker-Timestamp", tsStr)
	req.Header.Set("X-Worker-Signature", sig)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, res.StatusCode, nil
	}

	var job scheduler.CheckJob
	if err := json.NewDecoder(res.Body).Decode(&job); err != nil {
		return nil, res.StatusCode, err
	}

	return &job, res.StatusCode, nil
}

func ReportJob(serverURL, key string, h *health.HealthStatus) (int, error) {
	payload := map[string]interface{}{
		"worker_id":  h.Runner,
		"device_id":  h.DeviceID,
		"status":     h.Status,
		"latency_ms": h.Latency,
		"last_check": h.LastCheck,
	}

	body, _ := json.Marshal(payload)
	path := "/internal/worker/jobs/report"
	ts := time.Now().Unix()

	id, tsStr, sig := sign(key, h.Runner, http.MethodPost, path, body, ts)

	req, _ := http.NewRequest(http.MethodPost, serverURL+path, bytes.NewReader(body))
	req.Header.Set("X-Worker-Id", id)
	req.Header.Set("X-Worker-Timestamp", tsStr)
	req.Header.Set("X-Worker-Signature", sig)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	return res.StatusCode, nil
}
