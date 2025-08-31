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
	authMiddleware := middleware.NewAuthMiddleware(authSvc)

	options := ChiServerOptions{
		BaseRouter: chi.NewRouter(),
		Middlewares: map[string][]MiddlewareFunc{
			"common":                          {gzipMiddleware, logMiddleware},
			"GET /api/user/balance":           {authMiddleware},
			"POST /api/user/balance/withdraw": {authMiddleware},
			"GET /api/user/orders":            {authMiddleware},
			"POST /api/user/orders":           {authMiddleware},
			"GET /api/user/withdrawals":       {authMiddleware},
		},
	}

	return Handler(apiServer, options)
}
