package router

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

var startTime = time.Now()

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service,omitempty"`
	Version   string `json:"version,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
	Timestamp string `json:"timestamp"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:    "ok",
		Service:   os.Getenv("SERVICE_NAME"),
		Version:   os.Getenv("APP_VERSION"),
		Uptime:    time.Since(startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
