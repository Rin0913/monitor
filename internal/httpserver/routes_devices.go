package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/Rin0913/monitor/internal/device"
)

type addDeviceRequest struct {
    Address     string  `json:"address"`
    CheckMethod *string `json:"check_method"`
    IntervalSec *int    `json:"interval_sec"`
}

func (s *Server) addDevice(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    var req addDeviceRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }

    if req.Address == "" {
        http.Error(w, "missing address", http.StatusBadRequest)
        return
    }

    checkMethod := "tcp_check"
    if req.CheckMethod != nil {
        if *req.CheckMethod == "" {
            http.Error(w, "check_method cannot be empty", http.StatusBadRequest)
            return
        }
        checkMethod = *req.CheckMethod
    }

    interval := 10
    if req.IntervalSec != nil {
        if *req.IntervalSec <= 0 {
            http.Error(w, "interval_sec must be > 0", http.StatusBadRequest)
            return
        }
        interval = *req.IntervalSec
    }

    d := &device.Device{
        Address:     req.Address,
        Name:        req.Address,
        CheckMethod: checkMethod,
        IntervalSec: interval,
    }

    if err := s.deviceRepo.Save(r.Context(), d); err != nil {
        http.Error(w, "failed to save device", http.StatusInternalServerError)
        return
    }

    s.scheduler.Add(d)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(d)
}

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	devices, err := s.deviceRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list devices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(devices)
}

func (s *Server) getDeviceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	dev, err := s.deviceRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to get device", http.StatusInternalServerError)
		return
	}
	if dev == nil {
		http.NotFound(w, r)
		return
	}

	h, err := s.healthRepo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to get health", http.StatusInternalServerError)
		return
	}
	if h == nil {
		resp := map[string]interface{}{
			"status":     "unknown",
			"latency_ms": -1,
			"last_check": "unknown",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h)
}

func (s *Server) registerDeviceRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /devices", s.addDevice)
	mux.HandleFunc("GET /devices", s.listDevices)
	mux.HandleFunc("GET /devices/{id}", s.getDeviceStatus)
}

