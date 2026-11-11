package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "github.com/pikipici/informatika-monitoring/backend/internal/http"
)

type readinessPinger struct {
	err error
}

func (p readinessPinger) Ping(context.Context) error {
	return p.err
}

func TestReadyzReturnsServiceUnavailableWithoutDatabaseDependency(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	httpapi.NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	assertReadinessBody(t, recorder, "unavailable", "unavailable")
}

func TestReadyzReturnsOKForHealthyDatabase(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	httpapi.NewRouterWithPinger(readinessPinger{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	assertReadinessBody(t, recorder, "ok", "ok")
}

func TestReadyzReturnsSafeServiceUnavailableForDatabaseFailure(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	httpapi.NewRouterWithPinger(readinessPinger{err: errors.New("password=super-secret")}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	assertReadinessBody(t, recorder, "unavailable", "unavailable")
	if body := recorder.Body.String(); body == "" || containsSecret(body) {
		t.Fatalf("unsafe readiness body: %q", body)
	}
}

func TestHealthzStaysHealthyWhenDatabaseFails(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	httpapi.NewRouterWithPinger(readinessPinger{err: errors.New("database down")}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func assertReadinessBody(t *testing.T, recorder *httptest.ResponseRecorder, status, database string) {
	t.Helper()
	var response struct {
		Status   string `json:"status"`
		Database string `json:"database"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != status {
		t.Fatalf("status body = %q, want %q", response.Status, status)
	}
	if response.Database != database {
		t.Fatalf("database body = %q, want %q", response.Database, database)
	}
}

func containsSecret(body string) bool {
	for _, value := range []string{"super-secret", "password=", "postgres://"} {
		if len(body) >= len(value) {
			for i := 0; i+len(value) <= len(body); i++ {
				if body[i:i+len(value)] == value {
					return true
				}
			}
		}
	}
	return false
}
