//go:generate go tool oapi-codegen -config ../../tools/oapi.yaml ../../specification.yaml

package main

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"log"
	"net"
	"net/http"
	"os/signal"
	"sufirmart/internal/accrual"
	"sufirmart/internal/api/handler"
	"sufirmart/internal/config"
	"sufirmart/internal/db"
	"sufirmart/internal/dependencies"
	"sufirmart/internal/logger"
	"sufirmart/internal/order"
	"sufirmart/internal/repository"
	"sufirmart/internal/tools/workerpool"
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

	mainHandler := handler.InitApi(c)

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

		wp := c.WorkerPool()
		if wp != nil {
			wp.Start()
			c.Logger().Info("worker pool started")

			ordersRepo := repository.NewOrderRepository(c.Db(), c.Logger())
			accountsRepo := repository.NewAccountRepository(c.Db(), c.Logger())
			proc := order.NewProcessor(ordersRepo, accountsRepo, c.AccrualReader(), c.Logger(), c.Db(), wp)

			// Периодическое восстановление необработанных заказов
			ticker := time.NewTicker(5 * time.Second)
			go func() {
				defer ticker.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := proc.RecoverPending(ctx); err != nil {
							c.Logger().Error("periodic recovery failed", zap.Error(err))
						}
					}
				}
			}()
		}

		ln, err := net.Listen("tcp", server.Addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", server.Addr, err)
		}

		c.Logger().Info("server started", zap.String("addr", server.Addr))

		err = server.Serve(ln)
		if err != nil {
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

		wp := c.WorkerPool()
		if wp != nil {
			wp.Stop()
			c.Logger().Info("worker pool stopped")
		}

		err = server.Shutdown(shutdownTimeoutCtx)
		if err != nil {
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

	dbConnection, err := db.DBFactory(appConfig, appLogger)
	if err != nil {
		log.Fatal(err)
	}
	err = dbConnection.Ping()
	if err != nil {
		appLogger.Error("database ping failed", zap.Error(err))
		log.Fatal(err)
	}

	wp := workerpool.New(5, 100)
	wp.OnError(func(err error) {
		appLogger.Error("worker task error", zap.Error(err))
	})

	accrualReader := accrual.NewHttpReader(appConfig.AccuralAddress)

	deps := dependencies.NewContainer(
		appLogger,
		appConfig,
		dbConnection,
		accrualReader,
		wp,
	)

	return deps
}
