//go:generate go tool oapi-codegen -config ../../tools/oapi.yaml ../../specification.yaml

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"log"
	"net"
	"net/http"
	"os/signal"
	"sufirmart/internal/api"
	"sufirmart/internal/dependencies"
	"sufirmart/internal/logger"
	"syscall"
	"time"
)

const (
	timeoutServerShutdown = time.Second * 5
	timeoutShutdown       = time.Second * 10
)

func main() {
	c := InitContainer()

	if err := run(c); err != nil {
		c.Logger().Fatal("application error", zap.Error(err))
	}
}

func run(c *dependencies.Container) (err error) {
	rootCtx, cancelCtx := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
	defer cancelCtx()

	workGroup, ctx := errgroup.WithContext(rootCtx)

	// нештатное завершение программы по таймауту, если после завершения контекста
	// приложение не смогло завершиться за отведенный промежуток времени
	context.AfterFunc(ctx, func() {
		ctx, cancelCtx := context.WithTimeout(context.Background(), timeoutShutdown)
		defer cancelCtx()

		<-ctx.Done()
		c.Logger().Fatal("failed to gracefully shutdown the service")
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

		ln, err := net.Listen("tcp", server.Addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", server.Addr, err)
		}

		c.Logger().Info("server started", zap.String("addr", server.Addr))

		if err = server.Serve(ln); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}

			return fmt.Errorf("listen and server has failed: %w", err)
		}

		return nil
	})

	// graceful shutdown
	workGroup.Go(func() error {
		defer c.Logger().Info("server has been shutdown")

		<-ctx.Done()

		shutdownTimeoutCtx, cancelShutdownTimeoutCtx := context.WithTimeout(context.Background(), timeoutServerShutdown)
		defer cancelShutdownTimeoutCtx()

		if err := server.Shutdown(shutdownTimeoutCtx); err != nil {
			c.Logger().Error("an error occurred during server shutdown", zap.Error(err))
		}

		return nil
	})

	if err := workGroup.Wait(); err != nil {
		return err
	}

	return nil
}

func InitContainer() *dependencies.Container {
	cfg := zap.NewDevelopmentConfig()
	appLogger, err := logger.NewLogger(cfg)
	if err != nil {
		log.Fatal(err)
	}

	deps := dependencies.New(appLogger)

	return deps
}
