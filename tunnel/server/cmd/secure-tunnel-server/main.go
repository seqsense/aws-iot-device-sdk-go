package main

import (
	"log"
	"net/http"
	"time"

	"github.com/seqsense/aws-iot-device-sdk-go/v4/tunnel/server"
)

func main() {
	tunnelHandler := server.NewTunnelHandler()
	apiHandler := server.NewAPIHandler(tunnelHandler)
	mux := http.NewServeMux()
	mux.Handle("/", apiHandler)
	mux.Handle("/tunnel", tunnelHandler)

	s := &http.Server{
		Addr:         ":80",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	switch err := s.ListenAndServe(); err {
	case http.ErrServerClosed, nil:
	default:
		log.Fatal(err)
	}
}
