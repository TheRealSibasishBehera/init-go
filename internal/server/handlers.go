package server

import (
	"encoding/json"
	system "github.com/TheRealSibasishBehera/init-go/internal/system"
	"net/http"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "OK"}`))
}

func sysHandler(w http.ResponseWriter, r *http.Request) {
	sysInfo, err := system.CollectSystemInfo()
	if err != nil {
		http.Error(w, "Failed to collect system info", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	jsonData, err := json.Marshal(sysInfo)
	if err != nil {
		http.Error(w, "Failed to marshal system info", http.StatusInternalServerError)
		return
	}
	w.Write(jsonData)
}
