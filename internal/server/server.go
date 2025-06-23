package server

import (
	mux "github.com/gorilla/mux"
	"github.com/mdlayher/vsock"
	"net/http"
	"sync"
)

const (
	VSockPort = 1000
)

func NewAPIHandler(waitPidMutex *sync.Mutex, envs map[string]string) *APIHandler {
	return &APIHandler{
		waitPidMutex: waitPidMutex,
		envs:         envs,
	}
}

func StartVSocServer(waitPidMutex *sync.Mutex, envs map[string]string) {
	listener, err := vsock.Listen(VSockPort, nil)
	if err != nil {
		panic("Failed to start vsock listener: " + err.Error())
	}
	defer listener.Close()

	router := NewRouter()
	setupRoutes(router, waitPidMutex, envs)
	if err := http.Serve(listener, router); err != nil {
		panic("Failed to start HTTP server: " + err.Error())
	}
}

func setupRoutes(r *mux.Router, waitPidMutex *sync.Mutex, envs map[string]string) {
	r.HandleFunc("/status", statusHandler).Methods("GET")

	v1 := r.PathPrefix("/v1").Subrouter()
	setupAPIRoutes(v1, waitPidMutex, envs)
}

func setupAPIRoutes(r *mux.Router, waitPidMutex *sync.Mutex, envs map[string]string) {
	handler := NewAPIHandler(waitPidMutex, envs)

	r.HandleFunc("/sysinfo", sysHandler).Methods("GET")
	r.HandleFunc("/exec", handler.ExecHandler).Methods("POST")
	r.HandleFunc("/ws/exec", handler.WSExecHandler).Methods("GET")
}

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)
	return r
}
