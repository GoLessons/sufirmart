package testutil

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	validator "openapi.tanna.dev/go/validator/openapi3"
	"os"
	"strings"
	"sufirmart/internal/api"
	"sufirmart/internal/config"
	"sufirmart/internal/db"
	"sufirmart/internal/dependencies"
	"testing"
)

type Tester struct {
	Container     *dependencies.Container
	Router        http.Handler
	DB            *sql.DB
	OpenAPITester *openapi3.T
}

func NewTester(t *testing.T) *Tester {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URI")
	if dsn == "" {
		dsn = "postgresql://sufirmart:sufirmart@localhost:15432/sufirmart_test?sslmode=disable"
	}

	cfg := &config.AppConfig{DatabaseUri: dsn}
	logger := zap.NewNop()

	database, err := db.DBFactory(cfg)
	require.NoError(t, err)

	// Закрываем соединение после завершения теста
	t.Cleanup(func() { _ = database.Close() })

	migr := db.NewMigrator(database, logger)
	require.NoError(t, migr.Up())

	c := dependencies.NewContainer(logger, cfg, database)
	router := api.InitApi(c)

	doc, loadErr := openapi3.NewLoader().LoadFromFile("../../../specification.yaml")
	require.NoError(t, loadErr, "failed to load OpenAPI spec from specification.yaml")

	return &Tester{
		Container:     c,
		Router:        router,
		DB:            database,
		OpenAPITester: doc,
	}
}

func (e *Tester) ResetDB(t *testing.T) {
	t.Helper()
	_, err := e.DB.Exec(`TRUNCATE TABLE "sufirmart"."auth", "sufirmart"."user" CASCADE`)
	require.NoError(t, err)
}

func (e *Tester) DoRequest(t *testing.T, method string, url string, payload any) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()

	var body *bytes.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(data)
	} else {
		body = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	e.Router.ServeHTTP(rr, req)
	return rr, req
}

func (e *Tester) ValidateOpenAPI(t *testing.T, rr *httptest.ResponseRecorder, req *http.Request) {
	t.Helper()
	v := validator.NewValidator(e.OpenAPITester)
	tv := v.ForTest(t, rr, req)
	tv.Validate(rr, req)
}

func AssertBearerToken(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	authorization := rr.Header().Get("Authorization")
	require.NotEmpty(t, authorization)
	require.True(t, strings.HasPrefix(authorization, "Bearer "))
	require.Greater(t, len(authorization), len("Bearer "))
}
