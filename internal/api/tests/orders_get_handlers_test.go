package tests

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/api/tests/testutil"
	"sufirmart/internal/domain"
	"testing"
	"time"
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

// Воспроизводит тест await_order_processed для отладки
func TestGetApiUserOrders_AwaitProcessed(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	userID := I.SeedUser(t, "alice", "pwd")
	token := I.SeedAuthToken(t, userID)

	orderNum := "79927398713"
	headersPost := map[string]string{
		"Content-Type":  "text/plain",
		"Authorization": "Bearer " + token,
	}
	rrPost, _ := I.DoRaw(t, http.MethodPost, "/api/user/orders", []byte(orderNum), headersPost)
	require.Equal(t, http.StatusAccepted, rrPost.Code)

	orderNumber, err := domain.NewOrderNumber(orderNum)
	require.NoError(t, err)
	expectedAccrual := 123.45
	I.AccrualReader.Set(domain.NewAccural(orderNumber, domain.AccrualStatusProcessed, expectedAccrual))

	I.Container.WorkerPool().Start()
	defer I.Container.WorkerPool().Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	headersGet := map[string]string{
		"Authorization": "Bearer " + token,
	}

	for {
		select {
		case <-ctx.Done():
			t.Errorf("Не удалось дождаться окончания расчета начисления")
			return
		case <-ticker.C:
			rr, req := I.DoRaw(t, http.MethodGet, "/api/user/orders", nil, headersGet)

			require.Containsf(t, []int{http.StatusOK, http.StatusNoContent}, rr.Code,
				"Несоответствие статус кода ответа ожидаемому в хендлере '%s %s'", req.Method, req.URL.String())

			// Если пока 204 — продолжаем ждать
			if rr.Code == http.StatusNoContent {
				continue
			}

			// Для 200 — должен быть JSON
			require.Containsf(t, rr.Header().Get("Content-Type"), "application/json",
				"Заголовок ответа Content-Type содержит несоответствующее значение")

			var orders []api.Order
			require.NoErrorf(t, json.Unmarshal(rr.Body.Bytes(), &orders),
				"Ошибка при попытке сделать запрос на получение статуса расчета начисления в системе лояльности")

			if len(orders) == 0 || orders[0].Status != api.PROCESSED {
				continue
			}

			o := orders[0]
			require.Equal(t, orderNum, o.Number, "Номер заказа не соответствует ожидаемому")
			require.InDelta(t, expectedAccrual, *o.Accrual, 0.0001, "Начисление за заказ не соответствует ожидаемому")
			return
		}
	}
}
