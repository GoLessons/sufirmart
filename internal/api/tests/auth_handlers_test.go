package tests

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"sufirmart/internal/api/tests/testutil"
	"testing"
)

func TestPostApiUserRegister_Success(t *testing.T) {
	I := testutil.NewTester(t)
	I.ResetDB(t)

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/register", map[string]string{
		"login":    "alice",
		"password": "pwd",
	})

	require.Equal(t, http.StatusOK, rr.Code)
	testutil.AssertBearerToken(t, rr)
	I.ValidateOpenAPI(t, rr, req)
}

func TestPostApiUserRegister_Conflict(t *testing.T) {
	I := testutil.NewTester(t)
	I.ResetDB(t)

	registerOKResp, registerOKReq := I.DoRequest(t, http.MethodPost, "/api/user/register", map[string]string{
		"login":    "alice",
		"password": "pwd",
	})
	require.Equal(t, http.StatusOK, registerOKResp.Code)
	I.ValidateOpenAPI(t, registerOKResp, registerOKReq)

	registerConflictResp, registerConflictReq := I.DoRequest(t, http.MethodPost, "/api/user/register", map[string]string{
		"login":    "alice",
		"password": "pwd",
	})
	require.Equal(t, http.StatusConflict, registerConflictResp.Code)
	I.ValidateOpenAPI(t, registerConflictResp, registerConflictReq)
}

func TestPostApiUserRegister_BadRequest(t *testing.T) {
	I := testutil.NewTester(t)
	I.ResetDB(t)

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/register", map[string]string{
		"login":    "",
		"password": "",
	})

	require.Equal(t, http.StatusBadRequest, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestPostApiUserLogin_Success(t *testing.T) {
	I := testutil.NewTester(t)
	I.ResetDB(t)

	registerResp, registerReq := I.DoRequest(t, http.MethodPost, "/api/user/register", map[string]string{
		"login":    "bob",
		"password": "pwd",
	})
	require.Equal(t, http.StatusOK, registerResp.Code)
	I.ValidateOpenAPI(t, registerResp, registerReq)

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/login", map[string]string{
		"login":    "bob",
		"password": "pwd",
	})
	I.ValidateOpenAPI(t, rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	testutil.AssertBearerToken(t, rr)
}

func TestPostApiUserLogin_Unauthorized(t *testing.T) {
	I := testutil.NewTester(t)
	I.ResetDB(t)

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/login", map[string]string{
		"login":    "unknown",
		"password": "pwd",
	})

	require.Equal(t, http.StatusUnauthorized, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}

func TestPostApiUserLogin_BadRequest(t *testing.T) {
	I := testutil.NewTester(t)
	I.ResetDB(t)

	rr, req := I.DoRequest(t, http.MethodPost, "/api/user/login", map[string]string{
		"login":    "",
		"password": "",
	})

	require.Equal(t, http.StatusBadRequest, rr.Code)
	I.ValidateOpenAPI(t, rr, req)
}
