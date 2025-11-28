package tests

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/api/tests/testutil"
	"testing"
	"time"
)

func TestGetApiUserWithdrawals_Unauthorized(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/withdrawals", nil, map[string]string{})
	require.Equal(t, http.StatusUnauthorized, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestGetApiUserWithdrawals_NoContent(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/withdrawals", nil, headers)
	require.Equal(t, http.StatusNoContent, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestGetApiUserWithdrawals_Ok_SortedAndFiltered(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	aliceID := I.SeedUser(t, "alice", "pwd")
	aliceToken := I.SeedAuthToken(t, aliceID)
	bobID := I.SeedUser(t, "bob", "pwd")

	loc := time.FixedZone("UTC+3", 3*3600)
	t1 := time.Date(2020, 12, 9, 16, 9, 57, 0, loc)
	t2 := time.Date(2020, 12, 10, 10, 0, 0, 0, loc)

	err := I.HaveInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"id":           "01890f2a-aaaa-7aaa-b111-aaaaaaaaaaaa",
		"user_id":      aliceID,
		"order_num":    "2377225624",
		"accrual":      0.00,
		"withdraw":     500.00,
		"status":       1, // domain.TransactionStatusProcessed
		"comment":      "alice withdraw 1",
		"processed_at": t1,
	})
	require.NoError(t, err)

	err = I.HaveInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"id":           "01890f2a-bbbb-7bbb-b222-bbbbbbbbbbbb",
		"user_id":      aliceID,
		"order_num":    "1112223334",
		"accrual":      0.00,
		"withdraw":     42.00,
		"status":       1, // domain.TransactionStatusProcessed
		"comment":      "alice withdraw 2",
		"processed_at": t2,
	})
	require.NoError(t, err)

	err = I.HaveInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"id":           "01890f2a-cccc-7ccc-b333-cccccccccccc",
		"user_id":      aliceID,
		"order_num":    "acc001",
		"accrual":      100.00,
		"withdraw":     0.00,
		"status":       1,
		"comment":      "processed accrual should be ignored",
		"processed_at": t2,
	})
	require.NoError(t, err)

	err = I.HaveInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"id":           "01890f2a-dddd-7ddd-b444-dddddddddddd",
		"user_id":      aliceID,
		"order_num":    "plan001",
		"accrual":      0.00,
		"withdraw":     77.00,
		"status":       0, // domain.TransactionStatusPlanned
		"comment":      "planned should be ignored",
		"processed_at": t2,
	})
	require.NoError(t, err)

	err = I.HaveInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"id":           "01890f2a-eeee-7eee-b555-eeeeeeeeeeee",
		"user_id":      aliceID,
		"order_num":    "canc001",
		"accrual":      0.00,
		"withdraw":     88.00,
		"status":       -1, // domain.TransactionStatusCanceled
		"comment":      "canceled should be ignored",
		"processed_at": t2,
	})
	require.NoError(t, err)

	err = I.HaveInDatabase(`"sufirmart"."transaction"`, map[string]interface{}{
		"id":           "01890f2a-ffff-7fff-b666-ffffffffffff",
		"user_id":      bobID,
		"order_num":    "9990001112",
		"accrual":      0.00,
		"withdraw":     1000.00,
		"status":       1,
		"comment":      "other user should be ignored",
		"processed_at": t2,
	})
	require.NoError(t, err)

	headers := map[string]string{
		"Authorization": "Bearer " + aliceToken,
	}
	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/withdrawals", nil, headers)
	require.Equal(t, http.StatusOK, rr.Code)

	var got []api.Withdrawal
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))

	require.Len(t, got, 2)

	require.Equal(t, "1112223334", got[0].Order)
	require.InDelta(t, 42.0, got[0].Sum, 0.0001)
	require.True(t, got[0].ProcessedAt.Equal(t2))

	require.Equal(t, "2377225624", got[1].Order)
	require.InDelta(t, 500.0, got[1].Sum, 0.0001)
	require.True(t, got[1].ProcessedAt.Equal(t1))

	I.ValidateOpenAPI(t, rr, req)
}

func TestGetApiUserWithdrawals_InternalServerError(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	// Симулируем ошибку БД
	_, err := I.DB.Exec(`ALTER TABLE "sufirmart"."transaction" RENAME TO "transaction_bak"`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = I.DB.Exec(`ALTER TABLE "sufirmart"."transaction_bak" RENAME TO "transaction"`)
	})

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/withdrawals", nil, headers)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}
