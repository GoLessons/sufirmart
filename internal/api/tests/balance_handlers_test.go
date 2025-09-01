package tests

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/api/tests/testutil"
	"testing"
)

func TestGetApiUserBalance_Unauthorized(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/balance", nil, map[string]string{})
	require.Equal(t, http.StatusUnauthorized, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestGetApiUserBalance_ZeroWhenNoTransactions(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/balance", nil, headers)
	require.Equal(t, http.StatusOK, rr.Code)

	var got api.Balance
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Equal(t, 0.0, got.Current)
	require.Equal(t, 0.0, got.Withdrawn)

	I.ValidateOpenAPI(t, rr, req)
}

func TestGetApiUserBalance_Ok_SumsProcessed(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	aliceID := I.SeedUser(t, "alice", "pwd")
	aliceToken := I.SeedAuthToken(t, aliceID)
	bobID := I.SeedUser(t, "bob", "pwd")

	seedTransactions(t, I, aliceID, bobID)

	headers := map[string]string{
		"Authorization": "Bearer " + aliceToken,
	}
	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/balance", nil, headers)
	require.Equal(t, http.StatusOK, rr.Code)

	var balance api.Balance
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &balance))

	require.Equal(t, balance.Current, 100.50-40.25)
	require.Equal(t, balance.Withdrawn, 40.25)

	I.ValidateOpenAPI(t, rr, req)
}

func seedTransactions(t *testing.T, I *testutil.Tester, userID1, userID2 string) {
	t.Helper()
	transactions := map[string]map[string]interface{}{
		"tx1": {
			"id":       "01890f2a-1111-7aaa-b111-111111111111",
			"user_id":  userID1,
			"order_id": "01890f2a-2222-7bbb-b222-222222222222",
			"accrual":  100.50,
			"withdraw": 0.00,
			"status":   1,
			"comment":  "processed accrual",
		},
		"tx2": {
			"id":       "01890f2a-3333-7ccc-b333-333333333333",
			"user_id":  userID1,
			"order_id": "01890f2a-4444-7ddd-b444-444444444444",
			"accrual":  0.00,
			"withdraw": 40.25,
			"status":   1,
			"comment":  "processed withdraw",
		},
		"tx3": {
			"id":       "01890f2a-5555-7eee-b555-555555555555",
			"user_id":  userID1,
			"order_id": "01890f2a-6666-7fff-b666-666666666666",
			"accrual":  999.00,
			"withdraw": 999.00,
			"status":   0,
			"comment":  "planned should be ignored",
		},
		"tx4": {
			"id":       "01890f2a-7777-7000-b777-777777777777",
			"user_id":  userID1,
			"order_id": "01890f2a-8888-7111-b888-888888888888",
			"accrual":  777.00,
			"withdraw": 0.00,
			"status":   -1,
			"comment":  "canceled should be ignored",
		},
		"tx5": {
			"id":       "01890f2a-9999-7222-b999-999999999999",
			"user_id":  userID2,
			"order_id": "01890f2a-aaaa-7333-baaa-aaaaaaaaaaaa",
			"accrual":  500.00,
			"withdraw": 0.00,
			"status":   1,
			"comment":  "other user should be ignored",
		},
	}

	for name, row := range transactions {
		err := I.HaveInDatabase(`"sufirmart"."transaction"`, row)
		require.NoErrorf(t, err, "failed to seed transaction %s", name)
	}
}
