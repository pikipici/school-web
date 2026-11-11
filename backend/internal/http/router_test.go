package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "github.com/pikipici/informatika-monitoring/backend/internal/http"
)

func TestHealthzReturnsSafeJSONAndRequestID(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	httpapi.NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if got := recorder.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID is empty")
	}

	var response struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "ok" {
		t.Fatalf("status body = %q, want ok", response.Status)
	}
}

func TestPublicSummaryReturnsHonestZeroState(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/summary", nil)

	httpapi.NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Data struct {
			TotalGroups  int     `json:"totalGroups"`
			ManualUpdate bool    `json:"manualUpdate"`
			UpdatedAt    *string `json:"updatedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.TotalGroups != 0 {
		t.Fatalf("totalGroups = %d, want 0", response.Data.TotalGroups)
	}
	if !response.Data.ManualUpdate {
		t.Fatal("manualUpdate = false, want true")
	}
	if response.Data.UpdatedAt != nil {
		t.Fatalf("updatedAt = %q, want null", *response.Data.UpdatedAt)
	}
}

func TestUnknownAPIRouteReturnsJSONNotFrontend(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)

	httpapi.NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}

	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != "NOT_FOUND" {
		t.Fatalf("error code = %q, want NOT_FOUND", response.Error.Code)
	}
}
