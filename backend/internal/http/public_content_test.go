package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/content"
	httpapi "github.com/pikipici/informatika-monitoring/backend/internal/http"
)

type publicReaderStub struct{}

func (publicReaderStub) PublicAnnouncements(context.Context, content.Page) ([]content.PublicAnnouncement, int64, error) {
	return []content.PublicAnnouncement{{ID: 1, Title: "Info", Body: "Isi", PublishedAt: time.Unix(1, 0).UTC()}}, 1, nil
}
func (publicReaderStub) PublicAnnouncement(context.Context, int64) (content.PublicAnnouncement, error) {
	return content.PublicAnnouncement{}, content.ErrNotFound
}
func (publicReaderStub) PublicTopics(context.Context) ([]content.PublicTopic, error) {
	return []content.PublicTopic{{ID: 1, Code: "K1", Title: "Topik", SortOrder: 1}}, nil
}
func (publicReaderStub) PublicMaterials(context.Context, content.Page, string) ([]content.PublicMaterial, int64, error) {
	return []content.PublicMaterial{{ID: 2, Title: "Materi", Description: "Deskripsi", PublishedAt: time.Unix(2, 0).UTC()}}, 1, nil
}
func (publicReaderStub) PublicMaterial(context.Context, int64) (content.PublicMaterial, error) {
	return content.PublicMaterial{}, content.ErrNotFound
}

func TestPublicContentRoutesAndSafeErrors(t *testing.T) {
	r := httpapi.NewRouterWithDeps(httpapi.Deps{PublicContent: publicReaderStub{}})
	for _, target := range []string{"/api/v1/public/announcements", "/api/v1/public/topics", "/api/v1/public/materials?topic=K1"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != 200 {
			t.Fatalf("GET %s = %d: %s", target, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if json.Unmarshal(rec.Body.Bytes(), &body) != nil || body["data"] == nil {
			t.Fatalf("invalid envelope: %s", rec.Body.String())
		}
	}
	for _, target := range []string{"/api/v1/public/announcements/9", "/api/v1/public/materials/9"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != 404 || rec.Body.String() != "{\"error\":{\"code\":\"NOT_FOUND\",\"message\":\"Konten tidak ditemukan.\"}}\n" {
			t.Fatalf("unsafe 404 %s: (%d) %s", target, rec.Code, rec.Body.String())
		}
	}
}

func TestPublicPaginationValidationAndMethods(t *testing.T) {
	r := httpapi.NewRouterWithDeps(httpapi.Deps{PublicContent: publicReaderStub{}})
	for _, target := range []string{"/api/v1/public/announcements?page=x", "/api/v1/public/announcements?page=0", "/api/v1/public/materials?limit=101", "/api/v1/public/materials?limit=-1", "/api/v1/public/announcements?page=9223372036854775807&limit=100"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != 400 {
			t.Fatalf("GET %s = %d, want 400", target, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/public/topics", nil))
	if rec.Code != 405 {
		t.Fatalf("POST topics = %d", rec.Code)
	}
}

var _ content.PublicReader = publicReaderStub{}
