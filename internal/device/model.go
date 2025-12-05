package device

type Device struct {
	ID          string `json:"id"`
	Address     string `json:"address"`
	Name        string `json:"name"`
	CheckMethod string `json:"check_method"`
	IntervalSec int    `json:"interval_sec"`
}
