package tests

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"sufirmart/internal/api/tests/testutil"
	"sufirmart/internal/domain"
	"testing"
	"time"
)

func TestOrderProcessing_SuccessfulAccrual(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	orderNumber, err := domain.NewOrderNumber("79927398713")
	require.NoError(t, err)
	headers := map[string]string{
		"Content-Type":  "text/plain",
		"Authorization": "Bearer " + token,
	}
	rr, _ := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNumber.String()), headers)
	require.Equal(t, http.StatusAccepted, rr.Code)

	exists, err := I.SeeInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderNumber.String(),
		"status":    int16(domain.OrderStatusNew),
	})
	require.NoError(t, err)
	require.True(t, exists)

	accrualValue := 42.50
	accrual := domain.NewAccural(orderNumber, domain.AccrualStatusProcessed, accrualValue)
	I.AccrualReader.Set(accrual)

	// Запускаем обработку заказа вручную
	I.Container.WorkerPool().Start()
	defer I.Container.WorkerPool().Stop()
	time.Sleep(500 * time.Millisecond)

	exists, err = I.SeeInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"order_num": orderNumber.String(),
		"status":    int16(domain.OrderStatusProcessed),
	})
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = I.SeeInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderNumber.String(),
		"accrual":   accrualValue,
		"status":    int16(domain.TransactionStatusProcessed),
	})
	require.NoError(t, err)
	require.True(t, exists)
}

func TestOrderProcessing_InvalidOrder(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	orderNumber, err := domain.NewOrderNumber("79927398713")
	require.NoError(t, err)
	headers := map[string]string{
		"Content-Type":  "text/plain",
		"Authorization": "Bearer " + token,
	}
	rr, _ := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNumber.String()), headers)
	require.Equal(t, http.StatusAccepted, rr.Code)

	accrual := domain.NewAccural(orderNumber, domain.AccrualStatusInvalid, 0.0)
	I.AccrualReader.Set(accrual)

	// Запускаем обработку заказа вручную
	I.Container.WorkerPool().Start()
	defer I.Container.WorkerPool().Stop()
	time.Sleep(500 * time.Millisecond)

	exists, err := I.SeeInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"order_num": orderNumber.String(),
		"status":    int16(domain.OrderStatusInvalid),
	})
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = I.SeeInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderNumber.String(),
		"status":    int16(domain.TransactionStatusCanceled),
	})
	require.NoError(t, err)
	require.True(t, exists)
}
