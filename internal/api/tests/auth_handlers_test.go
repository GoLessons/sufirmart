package tests

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"sufirmart/internal/api/tests/testutil"
	"testing"
)

func TestPostApiUserRegister_Success(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/register", map[string]string{
		"login":    "alice",
		"password": "pwd",
	})

	require.Equal(t, http.StatusOK, rr.Code)
	token := I.AssertBearerToken(t, rr)
	I.ValidateOpenAPI(t, rr, req)

	existsUser, err := I.SeeInDatabase(`"sufirmart"."user"`, map[string]interface{}{"login": "alice"})
	require.NoError(t, err)
	require.True(t, existsUser)

	existsAuth, err := I.SeeInDatabase(`"sufirmart"."auth"`, map[string]interface{}{"token": token})
	require.NoError(t, err)
	require.True(t, existsAuth)
}

func TestPostApiUserRegister_Conflict(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	I.SeedUser(t, "alice", "pwd")

	registerConflictResp, registerConflictReq := I.DoRequest(t, http.MethodPost, "/api/user/register", map[string]string{
		"login":    "alice",
		"password": "pwd",
	})
	require.Equal(t, http.StatusConflict, registerConflictResp.Code)
	I.ValidateOpenAPI(t, registerConflictResp, registerConflictReq)
}

func TestPostApiUserRegister_BadRequest(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/register", map[string]string{
		"login":    "",
		"password": "",
	})

	require.Equal(t, http.StatusBadRequest, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestPostApiUserLogin_Success(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	I.SeedUser(t, "bob", "pwd")

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/login", map[string]string{
		"login":    "bob",
		"password": "pwd",
	})
	I.ValidateOpenAPI(t, rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	token := I.AssertBearerToken(t, rr)

	existsAuth, err := I.SeeInDatabase(`"sufirmart"."auth"`, map[string]interface{}{"token": token})
	require.NoError(t, err)
	require.True(t, existsAuth)
}

func TestPostApiUserLogin_Unauthorized(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/login", map[string]string{
		"login":    "unknown",
		"password": "pwd",
	})

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestPostApiUserLogin_BadRequest(t *testing.T) {
	I := testutil.NewTester(t)
	require.NoError(t, I.CleanDatabase("sufirmart"))

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/login", map[string]string{
		"login":    "",
		"password": "",
	})

	require.Equal(t, http.StatusBadRequest, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}
