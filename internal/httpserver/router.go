package httpserver

import (
    "fmt"
    "net/http"
)

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, "Hello!")
}

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", health)
}
