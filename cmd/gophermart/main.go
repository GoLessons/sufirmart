//go:generate go tool oapi-codegen -config ../../tools/oapi.yaml ../../specification.yaml

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sufirmart/internal/api"
	"sufirmart/internal/config"
	"sufirmart/internal/db"
	"sufirmart/internal/dependencies"
	"sufirmart/internal/logger"
	"sufirmart/internal/middleware"
	"syscall"
	"time"
)

const (
	timeoutServerShutdown = time.Second * 5
	timeoutShutdown       = time.Second * 10
)

func main() {
	c := InitContainer()

	tryMigrateDB(c.Config(), c.Db(), c.Logger())

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

	apiServer := api.Unimplemented{}
	router := chi.NewMux()
	logMiddleware := middleware.NewLoggingMiddleware(c.Logger())
	gzipMiddleware := middleware.NewGzipMiddleware()
	mainHandler := gzipMiddleware(logMiddleware(api.HandlerFromMux(apiServer, router)))

	server := &http.Server{
		Handler: mainHandler,
		Addr:    c.Config().ServerAddress,
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

	appConfig, cfgErr := config.LoadConfig(nil)
	if cfgErr != nil {
		log.Fatal("failed to load config: %w", cfgErr)
	}

	appLogger.Info("application started with config", zap.Any("config", appConfig))

	dbConnection, err := db.DBFactory(appConfig)
	if err != nil {
		log.Fatal(err)
	}
	err = dbConnection.Ping()
	if err != nil {
		appLogger.Error("database ping failed", zap.Error(err))
		log.Fatal(err)
	}

	deps := dependencies.NewContainer(appLogger, appConfig, dbConnection)

	return deps
}

func tryMigrateDB(cfg *config.AppConfig, dbConnection *sql.DB, serverLogger *zap.Logger) {
	if cfg.DatabaseUri != "" {
		migrator := db.NewMigrator(dbConnection, serverLogger)
		err := migrator.Up()
		if err != nil {
			serverLogger.Error("migrations error", zap.Error(err))
			os.Exit(1)
		}
	}
}
