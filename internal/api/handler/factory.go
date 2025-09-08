package handler

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/auth"
	"sufirmart/internal/dependencies"
	"sufirmart/internal/middleware"
	"sufirmart/internal/order"
	"sufirmart/internal/repository"
	"sufirmart/internal/user"
)

func InitApi(c *dependencies.Container) http.Handler {
	authSvc := auth.NewAuthService(c.Db(), c.Logger())
	userSvc := user.NewUserService(c.Db(), c.Logger())
	ordersRepo := repository.NewOrderRepository(c.Db(), c.Logger())
	accountsRepo := repository.NewAccountRepository(c.Db(), c.Logger())
	orderProcessor := order.NewProcessor(
		ordersRepo,
		accountsRepo,
		c.AccrualReader(),
		c.Logger(),
		c.Db(),
		c.WorkerPool(),
	)

	apiHandlers := map[string]http.HandlerFunc{
		"GetApiUserBalance":          NewGetApiUserBalanceHandler(accountsRepo),
		"PostApiUserBalanceWithdraw": NewPostApiUserBalanceWithdrawHandler(accountsRepo, c.Db()),
		"PostApiUserLogin":           NewPostApiUserLoginHandler(authSvc),
		"GetApiUserOrders":           NewGetApiUserOrdersHandler(ordersRepo),
		"PostApiUserOrders":          NewPostApiUserOrdersHandler(ordersRepo, orderProcessor),
		"PostApiUserRegister":        NewPostApiUserRegisterHandler(userSvc, authSvc),
		"GetApiUserWithdrawals":      NewGetApiUserWithdrawalsHandler(accountsRepo),
	}

	apiServer := api.NewApi(apiHandlers)

	logMiddleware := middleware.NewLoggingMiddleware(c.Logger())
	gzipMiddleware := middleware.NewGzipMiddleware()
	authMiddleware := middleware.NewAuthMiddleware(authSvc)

	options := api.ChiServerOptions{
		BaseRouter: chi.NewRouter(),
		Middlewares: map[string][]api.MiddlewareFunc{
			"common":                          {gzipMiddleware, logMiddleware},
			"GET /api/user/balance":           {authMiddleware},
			"POST /api/user/balance/withdraw": {authMiddleware},
			"GET /api/user/orders":            {authMiddleware},
			"POST /api/user/orders":           {authMiddleware},
			"GET /api/user/withdrawals":       {authMiddleware},
		},
	}

	return api.Handler(apiServer, options)
}
