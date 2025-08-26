//go:generate go tool oapi-codegen -config ../../tools/oapi.yaml ../../specification.yaml

package main

import (
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"sufirmart/internal/api"
)

func main() {
	apiServer := api.Unimplemented{}

	router := chi.NewMux()

	mainHandler := api.HandlerFromMux(apiServer, router)

	server := &http.Server{
		Handler: mainHandler,
		Addr:    "0.0.0.0:8080",
	}

	log.Fatal(server.ListenAndServe())
}
