package order

import (
	"context"
	"database/sql"
	"go.uber.org/zap"
	"sufirmart/internal/accrual"
	"sufirmart/internal/domain"
	"sufirmart/internal/repository"
	"sufirmart/internal/tools/db"
	"sufirmart/internal/tools/repeater"
	"sufirmart/internal/tools/workerpool"
	"time"
)

type Processor struct {
	orders        *repository.OrderRepository
	account       *repository.AccountRepository
	accrualReader accrual.Reader
	logger        *zap.Logger
	db            *sql.DB
	wp            *workerpool.WorkerPool
}

func NewProcessor(
	ordersRepo *repository.OrderRepository,
	accountsRepo *repository.AccountRepository,
	accrualReader accrual.Reader,
	logger *zap.Logger,
	db *sql.DB,
	wp *workerpool.WorkerPool,
) *Processor {
	return &Processor{
		orders:        ordersRepo,
		account:       accountsRepo,
		accrualReader: accrualReader,
		logger:        logger,
		db:            db,
		wp:            wp,
	}
}

func (p *Processor) Process(ctx context.Context, number domain.OrderNumber) {
	p.logger.Info("adding order processing task to worker pool", zap.String("order_number", number.String()))

	if p.wp == nil {
		p.logger.Error("worker pool is nil", zap.String("order_number", number.String()))
		return
	}

	p.wp.Add(func() error {
		p.logger.Info("processing order task started", zap.String("order_number", number.String()))
		p.processOrder(ctx, number)
		return nil
	})
}

func (p *Processor) processOrder(ctx context.Context, number domain.OrderNumber) {
	order := p.touchOrder(ctx, number)
	if order == nil {
		return
	}

	transactionID, err := p.account.RegisterTransaction(
		ctx,
		order.UserID(),
		order.Number().String(),
		0.0, // пока не знаем сумму транзакции
		"Order accrual",
	)
	if err != nil {
		p.logger.Error("failed to register transaction",
			zap.Error(err),
			zap.String("order_number", number.String()))
		return
	}

	rep := repeater.NewRepeater(func(err error) {
		p.logger.Warn(
			"accrual request failed, retrying",
			zap.Error(err),
			zap.String("order_number", number.String()),
		)
	})
	strategy := NewAdaptiveStrategy(time.Second, 0)

	_, err = rep.Repeat(strategy, func() (any, error) {
		accrualInfo, err := p.accrualReader.Get(number.String())
		if err != nil {
			return nil, err
		}

		switch accrualInfo.Status() {
		case domain.AccrualStatusRegistered, domain.AccrualStatusProcessing:
			p.logger.Info(
				"accrual is still processing",
				zap.String("order_number", number.String()),
				zap.Int16("accrual_status", int16(accrualInfo.Status())),
			)
			return nil, ErrStillProcessing

		case domain.AccrualStatusInvalid:
			_ = p.failOrder(ctx, order, transactionID, "accrual marked order as invalid")
			return nil, nil

		case domain.AccrualStatusProcessed:
			err := p.successOrder(ctx, order, transactionID, accrualInfo.Value())
			if err != nil {
				return nil, err
			}
			return nil, nil

		default:
			_ = p.failOrder(ctx, order, transactionID, "unknown accrual status")
			p.logger.Error(
				"unknown accrual status, couldn't continue",
				zap.String("order_number", number.String()),
				zap.Int16("accrual_status", int16(accrualInfo.Status())),
			)
			return nil, nil
		}
	})
}

func (p *Processor) touchOrder(ctx context.Context, number domain.OrderNumber) *domain.Order {
	result, err := db.WrapTransaction(ctx, p.db, func(txCtx context.Context) (*domain.Order, error) {
		order, err := p.orders.GetByNumber(txCtx, number, true)
		if err != nil {
			p.logger.Error(
				"order not exists",
				zap.Error(err),
				zap.String("order_number", number.String()),
			)
			return nil, err
		}

		if !order.CanBeProcessed() {
			p.logger.Info("order cannot be processed",
				zap.String("order_number", order.Number().String()),
				zap.Int16("status", int16(order.Status())))
			return nil, sql.ErrNoRows
		}

		if err := order.ChangeStatus(domain.OrderStatusProcessing); err != nil {
			p.logger.Error(
				"failed to change order status",
				zap.Error(err),
				zap.String("order_number", order.Number().String()),
			)
			return nil, err
		}

		if err := p.orders.Save(txCtx, order); err != nil {
			p.logger.Error(
				"failed to save order",
				zap.Error(err),
				zap.String("order_number", order.Number().String()),
			)
			return nil, err
		}

		return order, nil
	})

	if err != nil {
		p.logger.Error(
			"transaction failed",
			zap.Error(err),
		)
	}

	return result
}

func (p *Processor) failOrder(ctx context.Context, order *domain.Order, transactionID string, reason string) error {
	_, err := db.WrapTransaction(ctx, p.db, func(txCtx context.Context) (any, error) {
		err := order.ChangeStatus(domain.OrderStatusInvalid)
		if err != nil {
			p.logger.Error(
				"failed to change order status to invalid",
				zap.Error(err),
				zap.String("order_number", order.Number().String()),
			)
			return nil, err
		}

		err = p.orders.Save(txCtx, order)
		if err != nil {
			p.logger.Error(
				"failed to save order with invalid status",
				zap.Error(err),
				zap.String("order_number", order.Number().String()),
			)
			return nil, err
		}

		err = p.account.CancelTransaction(txCtx, transactionID, reason)
		if err != nil {
			p.logger.Error(
				"failed to cancel transaction",
				zap.Error(err),
				zap.String("transaction_id", transactionID),
			)
			return nil, err
		}

		p.logger.Warn(
			"order failed",
			zap.String("order_number", order.Number().String()),
			zap.String("reason", reason),
		)

		return nil, nil
	})

	return err
}

func (p *Processor) successOrder(ctx context.Context, order *domain.Order, transactionID string, accrualValue float64) error {
	_, err := db.WrapTransaction(ctx, p.db, func(txCtx context.Context) (bool, error) {
		err := order.ChangeStatus(domain.OrderStatusProcessed)
		if err != nil {
			p.logger.Error(
				"failed to change order status to processed",
				zap.Error(err),
				zap.String("order_number", order.Number().String()),
			)
			return false, err
		}

		err = p.orders.Save(txCtx, order)
		if err != nil {
			p.logger.Error("failed to save order with processed status",
				zap.Error(err),
				zap.String("order_number", order.Number().String()))
			return false, err
		}

		err = p.account.ApproveTransaction(txCtx, transactionID, accrualValue)
		if err != nil {
			p.logger.Error(
				"failed to approve transaction",
				zap.Error(err),
				zap.String("transaction_id", transactionID),
			)
			return false, err
		}

		p.logger.Info(
			"order processed successfully",
			zap.String("order_number", order.Number().String()),
			zap.Float64("accrual", accrualValue),
		)
		return true, nil
	})

	return err
}
