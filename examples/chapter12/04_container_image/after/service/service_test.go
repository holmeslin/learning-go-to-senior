package service_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/holmeslin/learning-go-to-senior/examples/chapter12/04_container_image/after/service"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name, method, target string
		wantCode             int
		wantBody             string
	}{
		{"livez", http.MethodGet, "/livez", http.StatusOK, "ok"},
		{"hello default", http.MethodGet, "/hello", http.StatusOK, "hello, world\n"},
		{"hello named", http.MethodGet, "/hello?name=gopher", http.StatusOK, "hello, gopher\n"},
		{"hello empty name", http.MethodGet, "/hello?name=", http.StatusOK, "hello, world\n"},
		{"wrong method", http.MethodPost, "/livez", http.StatusMethodNotAllowed, "Method Not Allowed\n"},
		{"unknown path", http.MethodGet, "/nope", http.StatusNotFound, "404 page not found\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			service.NewHandler().ServeHTTP(rec, httptest.NewRequest(tt.method, tt.target, nil))
			if rec.Code != tt.wantCode || rec.Body.String() != tt.wantBody {
				t.Errorf("%s %s = %d %q, want %d %q",
					tt.method, tt.target, rec.Code, rec.Body.String(), tt.wantCode, tt.wantBody)
			}
		})
	}
}
