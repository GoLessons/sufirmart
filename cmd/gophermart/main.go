//go:generate go tool oapi-codegen -config ../../tools/oapi.yaml ../../specification.yaml

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sufirmart/internal/api"
	"time"
)

const (
	timeoutServerShutdown = time.Second * 5
	timeoutShutdown       = time.Second * 10
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	rootCtx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelCtx()

	workGroup, ctx := errgroup.WithContext(rootCtx)

	// нештатное завершение программы по таймауту, если после завершения контекста
	// приложение не смогло завершиться за отведенный промежуток времени
	context.AfterFunc(ctx, func() {
		ctx, cancelCtx := context.WithTimeout(context.Background(), timeoutShutdown)
		defer cancelCtx()

		<-ctx.Done()
		log.Fatal("failed to gracefully shutdown the service")
	})

	// @todo init config

	apiServer := api.Unimplemented{}
	router := chi.NewMux()
	mainHandler := api.HandlerFromMux(apiServer, router)

	server := &http.Server{
		Handler: mainHandler,
		Addr:    "0.0.0.0:8080",
	}

	// server run
	workGroup.Go(func() (err error) {
		defer func() {
			errRec := recover()
			if errRec != nil {
				err = fmt.Errorf("a panic occurred: %v", errRec)
			}
		}()

		if err = server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			return fmt.Errorf("listen and server has failed: %w", err)
		}

		return nil
	})

	// graceful shutdown
	workGroup.Go(func() error {
		defer log.Print("server has been shutdown")

		<-ctx.Done()

		shutdownTimeoutCtx, cancelShutdownTimeoutCtx := context.WithTimeout(context.Background(), timeoutServerShutdown)
		defer cancelShutdownTimeoutCtx()

		if err := server.Shutdown(shutdownTimeoutCtx); err != nil {
			log.Printf("an error occurred during server shutdown: %v", err)
		}

		return nil
	})

	if err := workGroup.Wait(); err != nil {
		return err
	}

	return nil
}
