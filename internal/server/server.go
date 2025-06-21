package server

import (
	"net/http"

	mux "github.com/gorilla/mux"
	"github.com/mdlayher/vsock"
)

const (
	VSockPort = 1000
)

func StartServer() {
	listener, err := vsock.Listen(VSockPort, nil)
	if err != nil {
		panic("Failed to start vsock listener: " + err.Error())
	}
	defer listener.Close()

	router := NewRouter()
	setupRoutes(router)
	if err := http.Serve(listener, router); err != nil {
		panic("Failed to start HTTP server: " + err.Error())
	}
}

func setupRoutes(r *mux.Router) {
	r.HandleFunc("/status", statusHandler).Methods("GET")

	v1 := r.PathPrefix("/v1").Subrouter()
	setupAPIRoutes(v1)
}

func setupAPIRoutes(r *mux.Router) {
	r.HandleFunc("/sysinfo", sysHandler).Methods("GET")
	// r.HandleFunc("/exec", execHandler).Methods("GET")
	// r.HandleFunc("/ws", webSocketHandler).Methods("GET")
}

func NewRouter() *mux.Router {
	r := mux.NewRouter()
	r.StrictSlash(true)
	return r
}
