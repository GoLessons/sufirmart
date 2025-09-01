package tests

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/api/tests/testutil"
	"testing"
)

func TestGetApiUserOrders_Unauthorized(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/orders", nil, map[string]string{})
	require.Equal(t, http.StatusUnauthorized, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestGetApiUserOrders_NoContent(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/orders", nil, headers)
	require.Equal(t, http.StatusNoContent, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestGetApiUserOrders_Ok_SortedAndFiltered(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userAlice := I.SeedUser(t, "alice", "pwd")
	tokenAlice := I.SeedAuthToken(t, userAlice)
	userBob := I.SeedUser(t, "bob", "pwd")

	err := I.HaveInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"user_id":     userAlice,
		"order_num":   "79927398713",
		"status":      0,
		"uploaded_at": "2025-01-02T10:00:00Z",
	})
	require.NoError(t, err)
	err = I.HaveInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"user_id":     userAlice,
		"order_num":   "79927398715",
		"status":      1,
		"uploaded_at": "2025-01-02T12:00:00Z",
	})
	require.NoError(t, err)
	err = I.HaveInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"user_id":     userBob,
		"order_num":   "79927398716",
		"status":      3,
		"uploaded_at": "2025-01-02T11:00:00Z",
	})
	require.NoError(t, err)

	headers := map[string]string{
		"Authorization": "Bearer " + tokenAlice,
	}
	rr, req := I.DoRaw(t, http.MethodGet, "/api/user/orders", nil, headers)
	require.Equal(t, http.StatusOK, rr.Code)

	var got []api.Order
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	require.Len(t, got, 2)

	require.Equal(t, "79927398715", got[0].Number)
	require.Equal(t, "79927398713", got[1].Number)

	_, err = got[0].UploadedAt.MarshalText()
	require.NoError(t, err)
	_, err = got[1].UploadedAt.MarshalText()
	require.NoError(t, err)

	I.ValidateOpenAPI(t, rr, req)
}
