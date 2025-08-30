package api

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"sufirmart/internal/auth"
	"sufirmart/internal/dependencies"
	"sufirmart/internal/middleware"
	"sufirmart/internal/user"
)

func InitApi(c *dependencies.Container) http.Handler {
	// Инициализация сервисов
	authSvc := auth.NewAuthService(c.Db(), c.Logger())
	userSvc := user.NewUserService(c.Db(), c.Logger())

	apiServer := NewApi(authSvc, userSvc)

	logMiddleware := middleware.NewLoggingMiddleware(c.Logger())
	gzipMiddleware := middleware.NewGzipMiddleware()

	options := ChiServerOptions{
		BaseRouter: chi.NewRouter(),
		Middlewares: map[string][]MiddlewareFunc{
			"root": {gzipMiddleware, logMiddleware},
		},
	}

	return Handler(apiServer, options)
}
