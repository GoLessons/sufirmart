package tests

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"sufirmart/internal/api/tests/testutil"
	"sufirmart/internal/domain"
	"testing"
)

type withdrawReq struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

func TestPostApiUserBalanceWithdraw_Unauthorized(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	body, err := json.Marshal(withdrawReq{
		Order: "79927398713",
		Sum:   10.0,
	})
	require.NoError(t, err)

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	rr, req := I.DoRaw(t, http.MethodPost, "/api/user/balance/withdraw", body, headers)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestPostApiUserBalanceWithdraw_UnprocessableEntity_InvalidOrder(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	body, err := json.Marshal(withdrawReq{
		Order: "79927398714", // невалиден по Луну
		Sum:   10.0,
	})
	require.NoError(t, err)

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}
	rr, req := I.DoRaw(t, http.MethodPost, "/api/user/balance/withdraw", body, headers)

	require.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestPostApiUserBalanceWithdraw_PaymentRequired_InsufficientFunds(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	orderNum := "79927398713" // валидный номер
	sum := float32(10.0)

	body, err := json.Marshal(withdrawReq{
		Order: orderNum,
		Sum:   sum,
	})
	require.NoError(t, err)

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}
	rr, req := I.DoRaw(t, http.MethodPost, "/api/user/balance/withdraw", body, headers)

	require.Equal(t, http.StatusPaymentRequired, rr.Code)
	I.ValidateOpenAPI(t, rr, req)

	exists, err := I.SeeInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderNum,
		"withdraw":  float64(sum),
		"status":    domain.TransactionStatusCanceled,
	})
	require.NoError(t, err)
	require.True(t, exists)
}

func TestPostApiUserBalanceWithdraw_Ok_SufficientFunds(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	// Пополняем баланс за счет processed accrual
	err := I.HaveInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"id":        "01890f2a-aaaa-7aaa-b111-aaaaaaaaaaaa",
		"user_id":   userID,
		"order_num": "100500",
		"accrual":   100.50,
		"withdraw":  0.00,
		"status":    domain.TransactionStatusProcessed,
		"comment":   "seed accrual",
	})
	require.NoError(t, err)

	orderNum := "79927398713"
	sum := float32(40.25)

	body, err := json.Marshal(withdrawReq{
		Order: orderNum,
		Sum:   sum,
	})
	require.NoError(t, err)

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}
	rr, req := I.DoRaw(t, http.MethodPost, "/api/user/balance/withdraw", body, headers)

	require.Equal(t, http.StatusOK, rr.Code)
	I.ValidateOpenAPI(t, rr, req)

	// Должна появиться processed транзакция списания
	exists, err := I.SeeInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderNum,
		"withdraw":  float64(sum),
		"accrual":   0.0,
		"status":    domain.TransactionStatusProcessed,
	})
	require.NoError(t, err)
	require.True(t, exists)
}
