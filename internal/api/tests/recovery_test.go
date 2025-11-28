package tests

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sufirmart/internal/api/tests/testutil"
	"sufirmart/internal/domain"
	"sufirmart/internal/order"
	"sufirmart/internal/repository"
	"testing"
	"time"
)

func TestRecovery_ReprocessProcessingOrdersAndCleanupPlanned(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")

	orderProcessed := "79927398713" // будет PROCESSED
	orderInvalid := "2377225624"    // будет INVALID

	require.NoError(t, I.HaveInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderProcessed,
		"status":    int16(domain.OrderStatusProcessing),
	}))
	require.NoError(t, I.HaveInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderInvalid,
		"status":    int16(domain.OrderStatusProcessing),
	}))

	require.NoError(t, I.HaveInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"id":        uuid.New().String(),
		"user_id":   userID,
		"order_num": orderProcessed,
		"accrual":   0.0,
		"withdraw":  0.0,
		"status":    int16(domain.TransactionStatusPlanned),
		"comment":   "seed planned",
	}))
	require.NoError(t, I.HaveInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"id":        uuid.New().String(),
		"user_id":   userID,
		"order_num": orderInvalid,
		"accrual":   0.0,
		"withdraw":  0.0,
		"status":    int16(domain.TransactionStatusPlanned),
		"comment":   "seed planned",
	}))

	processedAccrual := 42.50
	accProcessed := domain.NewAccural(domain.OrderNumber(orderProcessed), domain.AccrualStatusProcessed, processedAccrual)
	accInvalid := domain.NewAccural(domain.OrderNumber(orderInvalid), domain.AccrualStatusInvalid, 0.0)
	I.AccrualReader.Set(accProcessed)
	I.AccrualReader.Set(accInvalid)

	I.Container.WorkerPool().Start()
	defer I.Container.WorkerPool().Stop()

	ordersRepo := repository.NewOrderRepository(I.DB, I.Container.Logger())
	accountsRepo := repository.NewAccountRepository(I.DB, I.Container.Logger())
	proc := order.NewProcessor(ordersRepo, accountsRepo, I.AccrualReader, I.Container.Logger(), I.DB, I.Container.WorkerPool())

	require.NoError(t, proc.RecoverPending(context.Background()))

	time.Sleep(800 * time.Millisecond)

	exists, err := I.SeeInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"order_num": orderProcessed,
		"status":    int16(domain.OrderStatusProcessed),
	})
	require.NoError(t, err)
	require.True(t, exists, "processed order must be PROCESSED")

	exists, err = I.SeeInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"order_num": orderInvalid,
		"status":    int16(domain.OrderStatusInvalid),
	})
	require.NoError(t, err)
	require.True(t, exists, "invalid order must be INVALID")

	notExists, err := I.DontSeeInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderProcessed,
		"status":    int16(domain.TransactionStatusPlanned),
	})
	require.NoError(t, err)
	require.True(t, notExists, "no PLANNED must remain for processed order")

	notExists, err = I.DontSeeInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderInvalid,
		"status":    int16(domain.TransactionStatusPlanned),
	})
	require.NoError(t, err)
	require.True(t, notExists, "no PLANNED must remain for invalid order")

	exists, err = I.SeeInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderProcessed,
		"accrual":   processedAccrual,
		"status":    int16(domain.TransactionStatusProcessed),
	})
	require.NoError(t, err)
	require.True(t, exists, "processed transaction with correct accrual must exist")

	exists, err = I.SeeInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderInvalid,
		"status":    int16(domain.TransactionStatusCanceled),
	})
	require.NoError(t, err)
	require.True(t, exists, "there must be a canceled transaction for invalid order")

	notExists, err = I.DontSeeInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderInvalid,
		"status":    int16(domain.TransactionStatusProcessed),
	})
	require.NoError(t, err)
	require.True(t, notExists, "there must be no processed transactions for invalid order")
}
