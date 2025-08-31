package tests

import (
	"net/http"
	"sufirmart/internal/api/tests/testutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostApiUserOrders_Unauthorized(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	orderNum := "79927398713"
	headers := map[string]string{
		"Content-Type": "text/plain",
	}
	rr, req := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNum), headers)

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestPostApiUserOrders_BadContentType(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	orderNum := "79927398713"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}
	rr, _ := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNum), headers)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	// Запрос не соответствует спецификации
}

func TestPostApiUserOrders_InvalidLuhn(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	orderNum := "79927398714"
	headers := map[string]string{
		"Content-Type":  "text/plain",
		"Authorization": "Bearer " + token,
	}
	rr, req := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNum), headers)

	require.Equal(t, http.StatusUnprocessableEntity, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestPostApiUserOrders_Accepted_NewOrder(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	orderNum := "79927398713"
	headers := map[string]string{
		"Content-Type":  "text/plain",
		"Authorization": "Bearer " + token,
	}
	rr, req := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNum), headers)

	require.Equal(t, http.StatusAccepted, rr.Code)
	I.ValidateOpenAPI(t, rr, req)

	exists, err := I.SeeInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderNum,
	})
	require.NoError(t, err)
	require.True(t, exists)
}

func TestPostApiUserOrders_Ok_ReuploadSameUser(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	orderNum := "79927398713"
	headers := map[string]string{
		"Content-Type":  "text/plain",
		"Authorization": "Bearer " + token,
	}

	rr1, req1 := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNum), headers)
	require.Equal(t, http.StatusAccepted, rr1.Code)
	I.ValidateOpenAPI(t, rr1, req1)

	rr2, req2 := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNum), headers)
	require.Equal(t, http.StatusOK, rr2.Code)
	I.ValidateOpenAPI(t, rr2, req2)

	exists, err := I.SeeInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"user_id":   userID,
		"order_num": orderNum,
	})
	require.NoError(t, err)
	require.True(t, exists)
}

func TestPostApiUserOrders_Conflict_OtherUser(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userAlice := I.SeedUser(t, "alice", "pwd")
	tokenAlice := I.SeedAuthToken(t, userAlice)
	userBob := I.SeedUser(t, "bob", "pwd")

	orderNum := "79927398713"
	err := I.HaveInDatabase(`"sufirmart"."order"`, map[string]interface{}{
		"user_id":   userBob,
		"order_num": orderNum,
		"status":    0, // OrderStatusNew
	})
	require.NoError(t, err)

	headers := map[string]string{
		"Content-Type":  "text/plain",
		"Authorization": "Bearer " + tokenAlice,
	}
	rr, req := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNum), headers)

	require.Equal(t, http.StatusConflict, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}
