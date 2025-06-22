package server

import (
	"encoding/json"
	"sync"
	"net/http"
	
	"github.com/TheRealSibasishBehera/init-go/internal/exec"
	system "github.com/TheRealSibasishBehera/init-go/internal/system"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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

type APIHandler struct {
	waitPidMutex *sync.Mutex
	envs         map[string]string
}

func (h *APIHandler) ExecHandler(w http.ResponseWriter, r *http.Request) {
	var req exec.ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate request
	if len(req.Cmd) == 0 {
		http.Error(w, "Command cannot be empty", http.StatusBadRequest)
		return
	}

	response, err := exec.ExecuteCommand(req, h.envs, h.waitPidMutex)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
