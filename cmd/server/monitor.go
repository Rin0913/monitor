package main

import (
	"log"
	"net/http"
	"time"
    "github.com/Rin0913/monitor/internal/httpserver"
)

func main() {
	mux := http.NewServeMux()
	
    httpserver.RegisterRoutes(mux)

    addr := ":8080"
	s := &http.Server{
		Addr:           addr,
		Handler:        mux,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

    log.Printf("listening on %s", addr)
    log.Fatal(s.ListenAndServe())
}
