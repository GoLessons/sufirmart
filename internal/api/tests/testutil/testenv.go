package testutil

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"net/http"
	"net/http/httptest"
	validator "openapi.tanna.dev/go/validator/openapi3"
	"os"
	"strings"
	"sufirmart/internal/api"
	"sufirmart/internal/config"
	"sufirmart/internal/db"
	"sufirmart/internal/dependencies"
	"sufirmart/internal/security"
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
	logger := NewTestLogger(t)

	database, err := db.DBFactory(cfg)
	require.NoError(t, err)

	// Закрываем соединение после завершения теста
	t.Cleanup(func() { _ = database.Close() })

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

	var bodyBytes []byte
	var body *bytes.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		require.NoError(t, err)
		bodyBytes = data
		body = bytes.NewReader(data)
	} else {
		bodyBytes = nil
		body = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	e.Router.ServeHTTP(rr, req)

	// Восстанавливаем тело запроса для OpenAPI-валидации
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	return rr, req
}

func (e *Tester) DoRaw(t *testing.T, method string, url string, body []byte, headers map[string]string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	e.Router.ServeHTTP(rr, req)
	req.Body = io.NopCloser(bytes.NewReader(body))
	return rr, req
}

func (e *Tester) HaveInDatabase(table string, data map[string]interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("data is empty")
	}

	columns := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))
	placeholders := []string{}
	i := 0
	for key, value := range data {
		i++
		columns = append(columns, key)
		values = append(values, value)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
	}
	columnsStr := strings.Join(columns, ", ")
	placeholdersStr := strings.Join(placeholders, ", ")

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, columnsStr, placeholdersStr)

	_, err := e.DB.Exec(query, values...)
	return err
}

func (e *Tester) SeeInDatabase(table string, criteria map[string]interface{}) (bool, error) {
	if len(criteria) == 0 {
		return false, fmt.Errorf("data is empty")
	}

	var conditions []string
	var values []interface{}
	i := 0
	for key, value := range criteria {
		i++
		conditions = append(conditions, fmt.Sprintf("%s = $%d", key, i))
		values = append(values, value)
	}

	query := fmt.Sprintf("SELECT true FROM %s WHERE %s LIMIT 1", table, strings.Join(conditions, " AND "))

	var exists bool
	err := e.DB.QueryRow(query, values...).Scan(&exists)

	return exists, err
}

func (e *Tester) DontSeeInDatabase(table string, criteria map[string]interface{}) (bool, error) {
	exists, err := e.SeeInDatabase(table, criteria)
	return !exists, err
}

func (e *Tester) CleanDatabase(schemaOrTable string) error {
	var query string
	if strings.Contains(schemaOrTable, ".") {
		query = fmt.Sprintf("TRUNCATE TABLE %s CASCADE", schemaOrTable)
	} else {
		_, err := e.DB.Exec(`
			CREATE OR REPLACE FUNCTION truncate_tables(schema IN VARCHAR) RETURNS void AS $$
				DECLARE
					statements CURSOR FOR
						SELECT tablename FROM pg_tables WHERE schemaname = quote_ident(schema) AND tablename NOT LIKE '%goose%';
				BEGIN
					FOR stmt IN statements LOOP
						EXECUTE 'TRUNCATE TABLE ' || quote_ident(schema) || '.' || quote_ident(stmt.tablename) || ' CASCADE;';
					END LOOP;
				END;
			$$ LANGUAGE plpgsql
        `)
		if err != nil {
			return err
		}
		query = fmt.Sprintf("SELECT truncate_tables('%s')", schemaOrTable)
	}

	_, err := e.DB.Exec(query)
	return err
}

func (e *Tester) ValidateOpenAPI(t *testing.T, rr *httptest.ResponseRecorder, req *http.Request) {
	t.Helper()
	v := validator.NewValidator(e.OpenAPITester)
	tv := v.ForTest(t, rr, req)
	tv.Validate(rr, req)
}

func (e *Tester) AssertBearerToken(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	authorization := rr.Header().Get("Authorization")
	require.NotEmpty(t, authorization)
	require.True(t, strings.HasPrefix(authorization, "Bearer "))
	require.Greater(t, len(authorization), len("Bearer "))
	return strings.TrimPrefix(authorization, "Bearer ")
}

func (e *Tester) SeedUser(t *testing.T, login, password string) string {
	t.Helper()
	u, err := uuid.NewV7()
	require.NoError(t, err)
	id := u.String()
	hash, err := security.PasswordHash(password)
	require.NoError(t, err)
	err = e.HaveInDatabase(`"sufirmart"."user"`, map[string]interface{}{
		"id":       id,
		"login":    login,
		"password": hash,
	})
	require.NoError(t, err)
	return id
}

func (e *Tester) SeedAuthToken(t *testing.T, userID string) string {
	t.Helper()
	token, err := security.GenerateToken(32)
	require.NoError(t, err)
	err = e.HaveInDatabase(`"sufirmart"."auth"`, map[string]interface{}{
		"user_id": userID,
		"token":   token,
	})
	require.NoError(t, err)
	return token
}

type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	w.t.Log(msg)
	return len(p), nil
}

func (w *testWriter) Sync() error {
	return nil
}

func NewTestLogger(t *testing.T) *zap.Logger {
	ws := zapcore.AddSync(&testWriter{t})
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(encoder, ws, zap.DebugLevel)
	return zap.New(core, zap.AddCaller())
}
