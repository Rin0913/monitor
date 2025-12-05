package health

import "time"

type HealthStatus struct {
	DeviceID  string                 `json:"device_id"`
	Status    string                 `json:"status"`
	Latency   int                    `json:"latency_ms"`
	LastCheck time.Time              `json:"last_check"`
	Runner    string                 `json:"runner"`
	Data      map[string]interface{} `json:"data,omitempty"`
}
